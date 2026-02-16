package ecommerce

// Links represents pagination links in API responses.
type Links struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
}

// IsLastPage returns true when there is no next page.
func (l *Links) IsLastPage() bool { return l.Next == "" }

// Meta represents pagination metadata in API responses.
type Meta struct {
	CurrentPage int `json:"current_page"`
	From        int `json:"from"`
	LastPage    int `json:"last_page"`
	PerPage     int `json:"per_page"`
	To          int `json:"to"`
	Total       int `json:"total"`
}

// --- Shop ---

type Shop struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type RootShops struct {
	Data  []Shop `json:"data"`
	Links Links  `json:"links"`
	Meta  Meta   `json:"meta"`
}

type RootShop struct {
	Data Shop `json:"data"`
}

// --- Product ---

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	URL         string  `json:"url"`
	ImageURL    string  `json:"image_url"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type RootProducts struct {
	Data  []Product `json:"data"`
	Links Links     `json:"links"`
	Meta  Meta      `json:"meta"`
}

type RootProduct struct {
	Data Product `json:"data"`
}

// --- Category ---

type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type RootCategories struct {
	Data  []Category `json:"data"`
	Links Links      `json:"links"`
	Meta  Meta       `json:"meta"`
}

type RootCategory struct {
	Data Category `json:"data"`
}

// --- Customer ---

type Customer struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type RootCustomers struct {
	Data  []Customer `json:"data"`
	Links Links      `json:"links"`
	Meta  Meta       `json:"meta"`
}

type RootCustomer struct {
	Data Customer `json:"data"`
}

// --- Order ---

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Order struct {
	ID         string      `json:"id"`
	CustomerID string      `json:"customer_id"`
	Status     string      `json:"status"`
	Total      float64     `json:"total"`
	Currency   string      `json:"currency"`
	Items      []OrderItem `json:"items"`
	CreatedAt  string      `json:"created_at"`
	UpdatedAt  string      `json:"updated_at"`
}

type RootOrders struct {
	Data  []Order `json:"data"`
	Links Links   `json:"links"`
	Meta  Meta    `json:"meta"`
}

type RootOrder struct {
	Data Order `json:"data"`
}

// --- Cart ---

type Cart struct {
	ID         string  `json:"id"`
	CustomerID string  `json:"customer_id"`
	Currency   string  `json:"currency"`
	Total      float64 `json:"total"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type RootCarts struct {
	Data  []Cart `json:"data"`
	Links Links  `json:"links"`
	Meta  Meta   `json:"meta"`
}

type RootCart struct {
	Data Cart `json:"data"`
}

// --- CartItem ---

type CartItem struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type RootCartItems struct {
	Data  []CartItem `json:"data"`
	Links Links      `json:"links"`
	Meta  Meta       `json:"meta"`
}

type RootCartItem struct {
	Data CartItem `json:"data"`
}

// --- Count (shared response) ---

type RootCount struct {
	Total int `json:"total"`
}
