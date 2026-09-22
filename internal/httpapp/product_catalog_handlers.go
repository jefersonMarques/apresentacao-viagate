package httpapp

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) productCatalogPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	showHidden := r.URL.Query().Get("show_hidden") == "1" || r.URL.Query().Get("show_deleted") == "1"
	openCategoryID := strings.TrimSpace(r.URL.Query().Get("open_category"))
	categories, err := a.catalogStore.ListAdmin(r.Context(), showHidden)
	if err != nil {
		a.logger.Error("load product catalog failed", "error", err)
		http.Error(w, "não foi possível carregar o catálogo de produtos", http.StatusInternalServerError)
		return
	}

	message := ""
	switch strings.TrimSpace(r.URL.Query().Get("saved")) {
	case "category":
		message = "Categoria salva."
	case "product":
		message = "Produto salvo."
	case "lifecycle":
		message = "Status do catálogo atualizado."
	case "product_deleted":
		message = "Produto excluído."
	case "product_archived":
		message = "O produto não pôde ser excluído porque está em uso ou é requisito de outro produto. Ele foi arquivado."
	case "product_reactivated":
		message = "Produto reativado."
	case "product_restored":
		message = "Produto restaurado."
	case "category_deleted":
		message = "Categoria excluída."
	case "category_archived":
		message = "A categoria não pôde ser excluída e foi arquivada."
	case "category_reactivated":
		message = "Categoria reativada."
	case "category_restored":
		message = "Categoria restaurada."
	}
	render(r.Context(), w, http.StatusOK, templates.ProductCatalogPage(
		user,
		categories,
		showHidden,
		openCategoryID,
		message,
		strings.TrimSpace(r.URL.Query().Get("error")),
	))
}

func (a *App) saveProductCategory(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	sortOrder, err := parseCatalogSortOrder(r.FormValue("sort_order"))
	if err != nil {
		redirectProductCatalog(w, r, false, "Ordem da categoria inválida.", "")
		return
	}
	id, err := a.catalogStore.SaveCategory(
		r.Context(),
		strings.TrimSpace(r.FormValue("id")),
		strings.TrimSpace(r.FormValue("name")),
		strings.TrimSpace(r.FormValue("description")),
		r.FormValue("is_active") == "1",
		sortOrder,
	)
	if err != nil {
		a.logger.Error("save product category failed", "user_id", user.ID, "category_id", r.FormValue("id"), "error", err)
		redirectProductCatalog(w, r, false, "Não foi possível salvar a categoria.", "")
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'catalog.category_saved','product_category',$2,'{}'::jsonb)
	`, user.ID, id)
	redirectProductCatalogWithOpenCategory(w, r, false, "", "category", id)
}

func (a *App) saveProductItem(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	categoryID := strings.TrimSpace(r.FormValue("category_id"))
	sortOrder, err := parseCatalogSortOrder(r.FormValue("sort_order"))
	if err != nil {
		redirectProductCatalogWithOpenCategory(w, r, false, "Ordem do produto inválida.", "", categoryID)
		return
	}
	dependencies := parseProductDependencyInputs(r.Form)
	id, err := a.catalogStore.SaveProduct(
		r.Context(),
		strings.TrimSpace(r.FormValue("id")),
		categoryID,
		strings.TrimSpace(r.FormValue("name")),
		strings.TrimSpace(r.FormValue("description")),
		strings.TrimSpace(r.FormValue("unit")),
		r.FormValue("is_active") == "1",
		sortOrder,
		dependencies,
	)
	if err != nil {
		a.logger.Error("save product item failed", "user_id", user.ID, "product_id", r.FormValue("id"), "error", err)
		message := "Não foi possível salvar o produto."
		switch {
		case errors.Is(err, catalog.ErrDependencyCycle):
			message = "A dependência cria um ciclo entre produtos. Revise as condicionais."
		case errors.Is(err, catalog.ErrInvalidDependency):
			message = "Existe uma dependência inválida. Use apenas produtos ativos e não arquivados."
		}
		redirectProductCatalogWithOpenCategory(w, r, false, message, "", categoryID)
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'catalog.product_saved','product',$2,'{}'::jsonb)
	`, user.ID, id)
	redirectProductCatalogWithOpenCategory(w, r, false, "", "product", categoryID)
}

func (a *App) updateProductLifecycle(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	productID := chi.URLParam(r, "id")
	action := strings.TrimSpace(r.FormValue("action"))
	showHidden := r.FormValue("show_hidden") == "1" || r.FormValue("show_deleted") == "1"

	err := a.catalogStore.UpdateProductLifecycle(r.Context(), productID, action)
	saved := productLifecycleSavedState(action)
	auditAction := action
	if errors.Is(err, catalog.ErrCatalogArchivedInstead) {
		err = nil
		saved = "product_archived"
		auditAction = "archive_fallback"
	}
	if err != nil {
		a.logger.Error("update product lifecycle failed", "user_id", user.ID, "product_id", productID, "action", action, "error", err)
		redirectProductCatalog(w, r, showHidden, catalogLifecycleError(err, "produto"), "")
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'catalog.product_lifecycle','product',$2,jsonb_build_object('action',$3::text))
	`, user.ID, productID, auditAction)
	redirectProductCatalog(w, r, showHidden, "", saved)
}

func (a *App) updateCategoryLifecycle(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	categoryID := chi.URLParam(r, "id")
	action := strings.TrimSpace(r.FormValue("action"))
	showHidden := r.FormValue("show_hidden") == "1" || r.FormValue("show_deleted") == "1"

	err := a.catalogStore.UpdateCategoryLifecycle(r.Context(), categoryID, action)
	saved := categoryLifecycleSavedState(action)
	auditAction := action
	if errors.Is(err, catalog.ErrCatalogArchivedInstead) {
		err = nil
		saved = "category_archived"
		auditAction = "archive_fallback"
	}
	if err != nil {
		a.logger.Error("update category lifecycle failed", "user_id", user.ID, "category_id", categoryID, "action", action, "error", err)
		redirectProductCatalog(w, r, showHidden, catalogLifecycleError(err, "categoria"), "")
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'catalog.category_lifecycle','product_category',$2,jsonb_build_object('action',$3::text))
	`, user.ID, categoryID, auditAction)
	redirectProductCatalog(w, r, showHidden, "", saved)
}

func productLifecycleSavedState(action string) string {
	switch action {
	case "delete":
		return "product_deleted"
	case "unarchive":
		return "product_reactivated"
	case "restore":
		return "product_restored"
	default:
		return "lifecycle"
	}
}

func categoryLifecycleSavedState(action string) string {
	switch action {
	case "delete":
		return "category_deleted"
	case "unarchive":
		return "category_reactivated"
	case "restore":
		return "category_restored"
	default:
		return "lifecycle"
	}
}

func parseCatalogSortOrder(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func parseProductDependencyInputs(form url.Values) []catalog.DependencyInput {
	indexes := []int{}
	modes := map[int]string{}
	for key, values := range form {
		if !strings.HasPrefix(key, "dependency_mode_") || len(values) == 0 {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(key, "dependency_mode_"))
		if err != nil {
			continue
		}
		indexes = append(indexes, index)
		modes[index] = strings.TrimSpace(values[0])
	}
	sort.Ints(indexes)

	result := make([]catalog.DependencyInput, 0, len(indexes))
	for _, index := range indexes {
		required := form["dependency_product_"+strconv.Itoa(index)]
		if len(required) == 0 {
			continue
		}
		result = append(result, catalog.DependencyInput{
			MatchMode:          modes[index],
			RequiredProductIDs: required,
		})
	}
	return result
}

func catalogLifecycleError(err error, resource string) string {
	switch {
	case errors.Is(err, catalog.ErrCatalogInUse):
		return "Não é possível excluir esta " + resource + " porque ela já foi usada em uma proposta."
	case errors.Is(err, catalog.ErrCatalogHasProducts):
		return "Não é possível excluir esta categoria enquanto houver produtos não excluídos nela."
	case errors.Is(err, catalog.ErrCatalogDependencyInUse):
		return "Não é possível excluir este produto porque ele é requisito de outro produto."
	case errors.Is(err, catalog.ErrCatalogNotFound):
		return "O item do catálogo não foi encontrado ou já foi alterado."
	default:
		return "Não foi possível atualizar o " + resource + "."
	}
}

func redirectProductCatalog(w http.ResponseWriter, r *http.Request, showHidden bool, errorMessage, saved string) {
	redirectProductCatalogWithOpenCategory(w, r, showHidden, errorMessage, saved, "")
}

func redirectProductCatalogWithOpenCategory(
	w http.ResponseWriter,
	r *http.Request,
	showHidden bool,
	errorMessage, saved, openCategoryID string,
) {
	values := url.Values{}
	if showHidden {
		values.Set("show_hidden", "1")
	}
	if errorMessage != "" {
		values.Set("error", errorMessage)
	}
	if saved != "" {
		values.Set("saved", saved)
	}
	if openCategoryID != "" {
		values.Set("open_category", openCategoryID)
	}
	target := "/admin/products"
	if encoded := values.Encode(); encoded != "" {
		target += "?" + encoded
	}
	if openCategoryID != "" {
		target += "#category-" + url.PathEscape(openCategoryID)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
