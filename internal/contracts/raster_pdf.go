package contracts

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	commercialRasterViewportWidth  = 1200
	commercialRasterViewportHeight = 900
	commercialRasterJPEGQuality    = 84
)

type commercialCaptureRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type rasterPDFPage struct {
	JPEG        []byte
	PixelWidth  int
	PixelHeight int
}

// RenderRasterizedDocument keeps the browser responsible only for visual
// rendering. Every commercial section is captured as one JPEG and the final
// PDF contains one image object per page, which makes scrolling substantially
// lighter than a browser-generated vector PDF.
func (r *PDFRenderer) RenderRasterizedDocument(ctx context.Context, document string) ([]byte, error) {
	if strings.TrimSpace(document) == "" {
		return nil, fmt.Errorf("commercial PDF HTML document is required")
	}

	browserPath, err := resolveBrowserExecutable(r.chromiumPath)
	if err != nil {
		return nil, err
	}

	dir, err := os.MkdirTemp("", "viagate-commercial-raster-*")
	if err != nil {
		return nil, fmt.Errorf("create commercial raster temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	htmlPath := filepath.Join(dir, "document.html")
	if err := os.WriteFile(htmlPath, []byte(document), 0o600); err != nil {
		return nil, fmt.Errorf("write commercial raster html: %w", err)
	}

	port, err := availableTCPPort()
	if err != nil {
		return nil, err
	}

	profileDir := filepath.Join(dir, "chrome-profile")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return nil, fmt.Errorf("create chrome profile dir: %w", err)
	}

	var browserLog bytes.Buffer
	command := exec.CommandContext(ctx, browserPath,
		"--headless",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--hide-scrollbars",
		"--no-first-run",
		"--no-default-browser-check",
		"--remote-allow-origins=*",
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port="+strconv.Itoa(port),
		"--user-data-dir="+profileDir,
		fmt.Sprintf("--window-size=%d,%d", commercialRasterViewportWidth, commercialRasterViewportHeight),
		"about:blank",
	)
	command.Stdout = &browserLog
	command.Stderr = &browserLog
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start browser for commercial raster PDF: %w", err)
	}
	defer func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		_ = command.Wait()
	}()

	webSocketURL, err := waitForDevToolsPage(ctx, port)
	if err != nil {
		return nil, fmt.Errorf("start browser DevTools: %w: %s", err, strings.TrimSpace(browserLog.String()))
	}

	ws, err := dialWebSocket(ctx, webSocketURL)
	if err != nil {
		return nil, fmt.Errorf("connect browser DevTools: %w", err)
	}
	defer ws.Close()
	client := &cdpClient{socket: ws}

	if err := client.call(ctx, "Page.enable", nil, nil); err != nil {
		return nil, err
	}
	if err := client.call(ctx, "Runtime.enable", nil, nil); err != nil {
		return nil, err
	}
	if err := client.call(ctx, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width":             commercialRasterViewportWidth,
		"height":            commercialRasterViewportHeight,
		"deviceScaleFactor": 1,
		"mobile":            false,
	}, nil); err != nil {
		return nil, err
	}

	if err := client.call(ctx, "Page.navigate", map[string]any{"url": fileURL(htmlPath)}, nil); err != nil {
		return nil, fmt.Errorf("navigate commercial raster page: %w", err)
	}
	if err := waitForCommercialCaptureReady(ctx, client); err != nil {
		return nil, err
	}

	rects, err := commercialCaptureLayout(ctx, client)
	if err != nil {
		return nil, err
	}
	if len(rects) == 0 {
		return nil, fmt.Errorf("commercial raster page has no visible sections")
	}

	pages := make([]rasterPDFPage, 0, len(rects))
	for index, rect := range rects {
		if rect.Width < 1 || rect.Height < 1 {
			return nil, fmt.Errorf("commercial section %d has invalid dimensions %.0fx%.0f", index+1, rect.Width, rect.Height)
		}

		var capture struct {
			Data string `json:"data"`
		}
		if err := client.call(ctx, "Page.captureScreenshot", map[string]any{
			"format":                "jpeg",
			"quality":               commercialRasterJPEGQuality,
			"fromSurface":           true,
			"captureBeyondViewport": true,
			"clip": map[string]any{
				"x":      rect.X,
				"y":      rect.Y,
				"width":  rect.Width,
				"height": rect.Height,
				"scale":  1,
			},
		}, &capture); err != nil {
			return nil, fmt.Errorf("capture commercial section %d: %w", index+1, err)
		}

		jpegBytes, err := base64.StdEncoding.DecodeString(capture.Data)
		if err != nil {
			return nil, fmt.Errorf("decode commercial section %d screenshot: %w", index+1, err)
		}
		config, err := jpeg.DecodeConfig(bytes.NewReader(jpegBytes))
		if err != nil {
			return nil, fmt.Errorf("inspect commercial section %d screenshot: %w", index+1, err)
		}
		pages = append(pages, rasterPDFPage{
			JPEG:        jpegBytes,
			PixelWidth:  config.Width,
			PixelHeight: config.Height,
		})
	}

	return buildRasterPDF(pages)
}

func availableTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve DevTools port: %w", err)
	}
	defer listener.Close()
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("resolve DevTools port")
	}
	return address.Port, nil
}

type devToolsPage struct {
	Type                 string `json:"type"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func waitForDevToolsPage(ctx context.Context, port int) (string, error) {
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/json/list", port)
	client := &http.Client{Timeout: 700 * time.Millisecond}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(8 * time.Second)
	defer timeout.Stop()

	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err == nil {
			response, requestErr := client.Do(request)
			if requestErr == nil {
				var pages []devToolsPage
				decodeErr := json.NewDecoder(response.Body).Decode(&pages)
				_ = response.Body.Close()
				if decodeErr == nil {
					for _, page := range pages {
						if page.Type == "page" && page.WebSocketDebuggerURL != "" {
							return page.WebSocketDebuggerURL, nil
						}
					}
				}
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout.C:
			return "", fmt.Errorf("DevTools page did not become available")
		case <-ticker.C:
		}
	}
}

type webSocketConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func dialWebSocket(ctx context.Context, rawURL string) (*webSocketConn, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "ws" {
		return nil, fmt.Errorf("unsupported DevTools websocket scheme %q", parsed.Scheme)
	}

	dialer := net.Dialer{Timeout: 4 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(connection)

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		_ = connection.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	requestURI := parsed.RequestURI()
	if requestURI == "" {
		requestURI = "/"
	}

	if _, err := fmt.Fprintf(connection,
		"GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		requestURI, parsed.Host, key,
	); err != nil {
		_ = connection.Close()
		return nil, err
	}

	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		_ = connection.Close()
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSwitchingProtocols {
		_ = connection.Close()
		return nil, fmt.Errorf("DevTools websocket handshake returned %s", response.Status)
	}

	hash := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	expectedAccept := base64.StdEncoding.EncodeToString(hash[:])
	if !strings.EqualFold(strings.TrimSpace(response.Header.Get("Sec-WebSocket-Accept")), expectedAccept) {
		_ = connection.Close()
		return nil, fmt.Errorf("DevTools websocket handshake validation failed")
	}

	return &webSocketConn{conn: connection, reader: reader}, nil
}

func (c *webSocketConn) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	_ = c.writeFrame(0x8, nil)
	return c.conn.Close()
}

func (c *webSocketConn) writeJSON(payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.writeFrame(0x1, data)
}

func (c *webSocketConn) writeFrame(opcode byte, payload []byte) error {
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}

	header := []byte{0x80 | opcode}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, 0x80|byte(length))
	case length <= 65535:
		header = append(header, 0x80|126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
	default:
		header = append(header, 0x80|127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	}
	header = append(header, mask...)

	masked := make([]byte, length)
	for index := range payload {
		masked[index] = payload[index] ^ mask[index%4]
	}

	if err := writeAll(c.conn, header); err != nil {
		return err
	}
	return writeAll(c.conn, masked)
}

func (c *webSocketConn) readMessage() ([]byte, error) {
	var message []byte
	started := false

	for {
		header := make([]byte, 2)
		if _, err := io.ReadFull(c.reader, header); err != nil {
			return nil, err
		}
		fin := header[0]&0x80 != 0
		opcode := header[0] & 0x0f
		masked := header[1]&0x80 != 0
		length := uint64(header[1] & 0x7f)

		switch length {
		case 126:
			extra := make([]byte, 2)
			if _, err := io.ReadFull(c.reader, extra); err != nil {
				return nil, err
			}
			length = uint64(binary.BigEndian.Uint16(extra))
		case 127:
			extra := make([]byte, 8)
			if _, err := io.ReadFull(c.reader, extra); err != nil {
				return nil, err
			}
			length = binary.BigEndian.Uint64(extra)
		}
		if length > 128<<20 {
			return nil, fmt.Errorf("DevTools websocket frame is too large")
		}

		mask := make([]byte, 4)
		if masked {
			if _, err := io.ReadFull(c.reader, mask); err != nil {
				return nil, err
			}
		}
		payload := make([]byte, int(length))
		if _, err := io.ReadFull(c.reader, payload); err != nil {
			return nil, err
		}
		if masked {
			for index := range payload {
				payload[index] ^= mask[index%4]
			}
		}

		switch opcode {
		case 0x8:
			return nil, io.EOF
		case 0x9:
			if err := c.writeFrame(0xA, payload); err != nil {
				return nil, err
			}
			continue
		case 0xA:
			continue
		case 0x1, 0x2:
			message = append(message[:0], payload...)
			started = true
		case 0x0:
			if !started {
				return nil, fmt.Errorf("unexpected DevTools websocket continuation frame")
			}
			message = append(message, payload...)
		default:
			continue
		}

		if fin && started {
			return message, nil
		}
	}
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return err
		}
		data = data[written:]
	}
	return nil
}

type cdpClient struct {
	socket *webSocketConn
	nextID int
}

type cdpEnvelope struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *cdpClient) call(ctx context.Context, method string, params any, result any) error {
	c.nextID++
	id := c.nextID
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.socket.conn.SetDeadline(deadline)
	} else {
		_ = c.socket.conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	request := map[string]any{"id": id, "method": method}
	if params != nil {
		request["params"] = params
	}
	if err := c.socket.writeJSON(request); err != nil {
		return fmt.Errorf("send DevTools %s: %w", method, err)
	}

	for {
		message, err := c.socket.readMessage()
		if err != nil {
			return fmt.Errorf("read DevTools %s: %w", method, err)
		}
		var envelope cdpEnvelope
		if err := json.Unmarshal(message, &envelope); err != nil || envelope.ID != id {
			continue
		}
		if envelope.Error != nil {
			return fmt.Errorf("DevTools %s failed (%d): %s", method, envelope.Error.Code, envelope.Error.Message)
		}
		if result != nil && len(envelope.Result) > 0 {
			if err := json.Unmarshal(envelope.Result, result); err != nil {
				return fmt.Errorf("decode DevTools %s result: %w", method, err)
			}
		}
		return nil
	}
}

type runtimeEvaluation struct {
	Result struct {
		Value json.RawMessage `json:"value"`
	} `json:"result"`
	ExceptionDetails json.RawMessage `json:"exceptionDetails"`
}

func evaluateRuntime(ctx context.Context, client *cdpClient, expression string, value any) error {
	var evaluation runtimeEvaluation
	if err := client.call(ctx, "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
	}, &evaluation); err != nil {
		return err
	}
	if len(evaluation.ExceptionDetails) > 0 && string(evaluation.ExceptionDetails) != "null" {
		return fmt.Errorf("browser evaluation failed: %s", string(evaluation.ExceptionDetails))
	}
	if value == nil {
		return nil
	}
	if len(evaluation.Result.Value) == 0 {
		return fmt.Errorf("browser evaluation returned no value")
	}
	return json.Unmarshal(evaluation.Result.Value, value)
}

func waitForCommercialCaptureReady(ctx context.Context, client *cdpClient) error {
	deadline := time.Now().Add(16 * time.Second)
	for {
		var ready bool
		err := evaluateRuntime(ctx, client, `document.documentElement.dataset.captureReady === '1'`, &ready)
		if err == nil && ready {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("wait for commercial capture layout: %w", err)
			}
			return fmt.Errorf("commercial capture layout timed out")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(120 * time.Millisecond):
		}
	}
}

func commercialCaptureLayout(ctx context.Context, client *cdpClient) ([]commercialCaptureRect, error) {
	expression := `(() => Array.from(document.querySelectorAll('.proposal-slide, #presentation > .slide'))
  .filter((slide) => getComputedStyle(slide).display !== 'none')
  .map((slide) => {
    const rect = slide.getBoundingClientRect();
    return {
      x: Math.max(0, rect.left + window.scrollX),
      y: Math.max(0, rect.top + window.scrollY),
      width: Math.ceil(rect.width),
      height: Math.ceil(rect.height)
    };
  }))()`
	var rects []commercialCaptureRect
	if err := evaluateRuntime(ctx, client, expression, &rects); err != nil {
		return nil, fmt.Errorf("measure commercial sections: %w", err)
	}
	return rects, nil
}

func buildRasterPDF(pages []rasterPDFPage) ([]byte, error) {
	if len(pages) == 0 {
		return nil, fmt.Errorf("raster PDF requires at least one page")
	}

	const pageWidthPoints = 595.276
	objectCount := 2 + len(pages)*3
	offsets := make([]int, objectCount+1)
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")

	writeObject := func(id int, body []byte) {
		offsets[id] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n", id)
		output.Write(body)
		output.WriteString("\nendobj\n")
	}

	writeObject(1, []byte("<< /Type /Catalog /Pages 2 0 R >>"))
	var kids strings.Builder
	for index := range pages {
		fmt.Fprintf(&kids, "%d 0 R ", 3+index*3)
	}
	writeObject(2, []byte(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(pages), kids.String())))

	for index, page := range pages {
		if page.PixelWidth <= 0 || page.PixelHeight <= 0 || len(page.JPEG) == 0 {
			return nil, fmt.Errorf("raster PDF page %d is invalid", index+1)
		}
		pageObject := 3 + index*3
		imageObject := pageObject + 1
		contentObject := pageObject + 2
		pageHeightPoints := pageWidthPoints * float64(page.PixelHeight) / float64(page.PixelWidth)

		pageBody := fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.3f %.3f] /Resources << /XObject << /Im0 %d 0 R >> >> /Contents %d 0 R >>",
			pageWidthPoints, pageHeightPoints, imageObject, contentObject,
		)
		writeObject(pageObject, []byte(pageBody))

		imageHeader := fmt.Sprintf(
			"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Interpolate true /Length %d >>\nstream\n",
			page.PixelWidth, page.PixelHeight, len(page.JPEG),
		)
		var imageBody bytes.Buffer
		imageBody.WriteString(imageHeader)
		imageBody.Write(page.JPEG)
		imageBody.WriteString("\nendstream")
		writeObject(imageObject, imageBody.Bytes())

		content := []byte(fmt.Sprintf("q\n%.3f 0 0 %.3f 0 0 cm\n/Im0 Do\nQ\n", pageWidthPoints, pageHeightPoints))
		contentBody := append([]byte(fmt.Sprintf("<< /Length %d >>\nstream\n", len(content))), content...)
		contentBody = append(contentBody, []byte("endstream")...)
		writeObject(contentObject, contentBody)
	}

	xrefOffset := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n", objectCount+1)
	output.WriteString("0000000000 65535 f \n")
	for id := 1; id <= objectCount; id++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", objectCount+1, xrefOffset)
	return output.Bytes(), nil
}
