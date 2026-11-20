// Package openrtb holds a simplified OpenRTB 2.5 subset.
package openrtb

type BidRequest struct {
	ID     string  `json:"id"`
	Imp    []Imp   `json:"imp"`
	App    *App    `json:"app,omitempty"`
	Device *Device `json:"device,omitempty"`
	User   *User   `json:"user,omitempty"`
	TMax   int     `json:"tmax,omitempty"` // ms
}

type Imp struct {
	ID       string  `json:"id"`
	Banner   *Banner `json:"banner,omitempty"`
	BidFloor float64 `json:"bidfloor,omitempty"`
}

type Banner struct {
	W int `json:"w"`
	H int `json:"h"`
}

type App struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Bundle string `json:"bundle,omitempty"`
}

type Device struct {
	IP  string `json:"ip,omitempty"`
	UA  string `json:"ua,omitempty"`
	OS  string `json:"os,omitempty"`
	IFA string `json:"ifa,omitempty"`
}

type User struct {
	ID string `json:"id"`
}

type BidResponse struct {
	ID      string    `json:"id"`
	SeatBid []SeatBid `json:"seatbid,omitempty"`
}

type SeatBid struct {
	Bid  []Bid  `json:"bid"`
	Seat string `json:"seat,omitempty"`
}

type Bid struct {
	ID    string  `json:"id"`
	ImpID string  `json:"impid"`
	Price float64 `json:"price"`
	AdM   string  `json:"adm,omitempty"`
	CRID  string  `json:"crid,omitempty"`
}
