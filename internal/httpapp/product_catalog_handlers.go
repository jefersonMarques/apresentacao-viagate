package httpapp

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) productCatalogPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	categories, err := a.catalogStore.ListAdmin(r.Context())
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
	}
	render(r.Context(), w, http.StatusOK, templates.ProductCatalogPage(
		user,
		categories,
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
		http.Redirect(w, r, "/admin/products?error="+queryEscape("Ordem da categoria inválida."), http.StatusSeeOther)
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
		http.Redirect(w, r, "/admin/products?error="+queryEscape("Não foi possível salvar a categoria."), http.StatusSeeOther)
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'catalog.category_saved','product_category',$2,'{}'::jsonb)
	`, user.ID, id)
	http.Redirect(w, r, "/admin/products?saved=category", http.StatusSeeOther)
}

func (a *App) saveProductItem(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	sortOrder, err := parseCatalogSortOrder(r.FormValue("sort_order"))
	if err != nil {
		http.Redirect(w, r, "/admin/products?error="+queryEscape("Ordem do produto inválida."), http.StatusSeeOther)
		return
	}
	id, err := a.catalogStore.SaveProduct(
		r.Context(),
		strings.TrimSpace(r.FormValue("id")),
		strings.TrimSpace(r.FormValue("category_id")),
		strings.TrimSpace(r.FormValue("name")),
		strings.TrimSpace(r.FormValue("description")),
		strings.TrimSpace(r.FormValue("unit")),
		r.FormValue("is_active") == "1",
		sortOrder,
	)
	if err != nil {
		a.logger.Error("save product item failed", "user_id", user.ID, "product_id", r.FormValue("id"), "error", err)
		http.Redirect(w, r, "/admin/products?error="+queryEscape("Não foi possível salvar o produto."), http.StatusSeeOther)
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'catalog.product_saved','product',$2,'{}'::jsonb)
	`, user.ID, id)
	http.Redirect(w, r, "/admin/products?saved=product", http.StatusSeeOther)
}

func parseCatalogSortOrder(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}
