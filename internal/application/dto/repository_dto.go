package dto

// RepositoryResponse represents repository data in API responses
type RepositoryResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	FullName    string  `json:"full_name"`
	Description *string `json:"description"`
	URL         string  `json:"url"`
	HTMLURL     *string `json:"html_url"`
	Private     bool    `json:"private"`
	Fork        bool    `json:"fork"`
	Stars       int32   `json:"stars"`
	Watchers    int32   `json:"watchers"`
	Forks       int32   `json:"forks"`
	Language    *string `json:"language"`
	CreatedAt   string  `json:"created_at"`
}

// RepositoryListResponse represents a paginated list of repositories
type RepositoryListResponse struct {
	Repositories []*RepositoryResponse `json:"repositories"`
	Pagination   PaginationResponse    `json:"pagination"`
}
