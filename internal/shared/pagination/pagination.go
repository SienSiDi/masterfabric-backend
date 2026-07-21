package pagination

import "net/http"

const MaxPage = 1_000_000

type Params struct {
	Page  int
	Limit int
}

func Parse(r *http.Request, defaultLimit int) Params {
	q := r.URL.Query()
	page := 1
	limit := defaultLimit
	if v := q.Get("page"); v != "" {
		if n := atoiSafe(v); n > 0 {
			page = n
		}
	}
	if v := q.Get("limit"); v != "" {
		if n := atoiSafe(v); n > 0 && n <= 100 {
			limit = n
		}
	}
	if page > MaxPage {
		page = MaxPage
	}
	return Params{Page: page, Limit: limit}
}

func (p Params) Offset() int { return (p.Page - 1) * p.Limit }

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
		if n > MaxPage*100 {
			return 0
		}
	}
	return n
}
