package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// Create a type for the message
type messageData struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Create a type for the http payload
type RequestData struct {
	Messages []messageData `json:"messages"`
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
}

type ResponseData struct {
	Model      string      `json:"model"`
	Created_at string      `json:"created_at"`
	Message    messageData `json:"message"`
	Done       bool        `json:"done"`
}

func main() {
	url := "http://localhost:11434/api/chat"
	defaultModel := "llama3.1"

	app := tview.NewApplication()

	// Creating UI elements
	output := tview.NewTextView().SetText("").SetDynamicColors(true)
	header := tview.NewTextView().SetText("Model: " + defaultModel).SetTextAlign(tview.AlignCenter).SetDynamicColors(true)
	input := tview.NewInputField().SetLabel("Prompt: ").SetFieldWidth(50).SetFieldBackgroundColor(tcell.ColorBlack).SetFieldTextColor(tcell.ColorWhite)

	// Grid layout
	grid := tview.NewGrid().
		SetRows(3, 0, 3). // Header (fixed 3 rows), Output (expandable), Input (fixed 3 rows)
		SetColumns(0).    // Single full-width column
		SetBorders(true)

	// Adding components to grid
	grid.AddItem(header, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(output, 1, 0, 1, 1, 0, 0, false)
	grid.AddItem(input, 2, 0, 1, 1, 0, 0, true) // Make input active

	// var mu sync.Mutex // Mutex to synchronize access to the output field
	getLLMResponse := func(inputText string) {

		payload := RequestData{Messages: []messageData{
			{Role: "user", Content: inputText},
		},
			Model:  defaultModel,
			Stream: false}

		// Convert payload to JSON
		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Fatalf("Error marshalling JSON: %v", err)
		}
		// Create a new HTTP POST request
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}
		// Set the content type to application/json
		req.Header.Set("Content-Type", "application/json")

		// Make the HTTP request
		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			log.Fatalf("Error making request: %v", err)
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error reading response: %v", err)
		}

		var response ResponseData
		err = json.Unmarshal(body, &response)
		if err != nil {
			log.Fatalf("Error unmarshalling response: %v", err)
		}
		// Placeholder for model output
		formattedText := output.GetText(false) + "\n" + "[blue]----------- Model output -----------[white]" + "\n" + response.Message.Content
		output.SetText(formattedText)
		app.Draw() // Refresh the UI
	}

	// Input handling (press Enter to display input in output field)
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			promptText := input.GetText()
			if promptText == "" {
				return
			}

			currentText := output.GetText(false)
			formattedText := currentText + "\n" + "[green]----------- User input -----------[white]" + "\n" + input.GetText()
			output.SetText(formattedText)
			input.SetText("") // Clear input field

			// Call the model
			go getLLMResponse(promptText)
		}
	})
	// Run the application
	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}
}
