package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Departure struct {
	When      time.Time `json:"when"`
	Direction string    `json:"direction"`
	Line      struct {
		Name string `json:"name"`
	} `json:"line"`
	Delay *int `json:"delay"`
}

type DepartureResponse struct {
	Departures []Departure `json:"departures"`
}

func main() {
	// Salzgitter Ostfalia ID
	url := "https://v6.db.transport.rest/stops/991604089/departures?duration=120&results=5"

	fmt.Println("Fetching Live VRB Transit Data from HAFAS...")

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var res DepartureResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	fmt.Println("\n--- 🚌 Next Departures: Ostfalia Salzgitter ---")
	for _, d := range res.Departures {
		delayStr := ""
		if d.Delay != nil && *d.Delay > 0 {
			delayStr = fmt.Sprintf(" (+%d min delay)", *d.Delay/60)
		}

		fmt.Printf("[%s] Line %s -> %s%s\n",
			d.When.Local().Format("15:04"),
			d.Line.Name,
			d.Direction,
			delayStr)
	}
}
