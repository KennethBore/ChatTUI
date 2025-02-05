package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// Add this helper function at the top level:
func colorToHex(color tcell.Color) string {
	r, g, b := color.RGB()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// MARK: Types -------------------------------------------------------------------
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

// MARK: Chat History ------------------------------------------------------------
func getChatHistory(chatFile string) ([]messageData, error) {
	chatHistory := []messageData{}
	jsonHistory, err := os.ReadFile(chatFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonHistory, &chatHistory)
	if err != nil {
		return nil, err
	}
	return chatHistory, nil
}

func saveChatHistory(chatFile string, chatHistory []messageData) error {
	jsonHistory, err := json.Marshal(chatHistory)
	if err != nil {
		return err
	}
	os.WriteFile(chatFile, jsonHistory, 0644)
	return nil
}

func clearChatHistory(chatFile string) error {
	return os.WriteFile(chatFile, []byte("[]"), 0644)
}

// MARK: Main --------------------------------------------------------------------
func main() {
	colorBackground := tcell.NewRGBColor(40, 42, 54)
	colorOutputBorder := tcell.NewRGBColor(189, 147, 249)
	colorInputBorder := tcell.NewRGBColor(139, 233, 253)
	colorHeaderBorder := tcell.NewRGBColor(80, 250, 123)
	colorInput := tcell.NewRGBColor(68, 71, 90)

	userColor := tcell.NewRGBColor(80, 250, 123)
	modelColor := tcell.NewRGBColor(255, 184, 108)

	chatFile := "chat.json"

	chatHistory, err := getChatHistory(chatFile)
	if err != nil {
		log.Fatalf("Error getting chat history: %v", err)
	}
	url := "http://localhost:11434/api/chat"
	defaultModel := "llama3.1"

	app := tview.NewApplication()

	// Creating UI elements
	output := tview.NewTextView().SetText("").SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	header := tview.NewTextView().SetText("").SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	input := tview.NewInputField().SetFieldWidth(50).SetFieldBackgroundColor(colorInput).SetFieldTextColor(tcell.ColorWhite).SetLabel("[white]Prompt: ").SetLabelStyle(tcell.StyleDefault.Background(colorInput).Bold(true))

	output.SetBackgroundColor(colorBackground).SetBorder(true).SetBorderColor(colorOutputBorder)
	header.SetBackgroundColor(colorBackground).SetBorder(true).SetBorderColor(colorHeaderBorder)
	input.SetBackgroundColor(colorBackground).SetBorder(true).SetBorderColor(colorInputBorder)
	// Grid layout
	grid := tview.NewGrid().
		SetRows(3, 0, 3). // Header (fixed 3 rows), Output (expandable), Input (fixed 3 rows)
		SetColumns(0)     // Single full-width column

	grid.SetBackgroundColor(colorBackground)

	// Adding components to grid
	grid.AddItem(header, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(output, 1, 0, 1, 1, 0, 0, false)
	grid.AddItem(input, 2, 0, 1, 1, 0, 0, true) // Make input active

	header.SetText("[" + colorToHex(tcell.ColorWhite) + "]Model: " + defaultModel)

	// MARK: LLM Response ----------------------------------------------------------
	getLLMResponse := func(inputText string) {
		chatHistory = append(chatHistory, messageData{Role: "user", Content: inputText})
		payload := RequestData{Messages: chatHistory,
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
			output.SetText(output.GetText(false) + "\n[red]Error: Could not connect to " + defaultModel + ". Is it running?[white]")
			output.ScrollToEnd()
			app.Draw()
			return
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
		formattedText := output.GetText(false) + "\n" + "[" + colorToHex(modelColor) + "]----------- Model output -----------[" + colorToHex(colorOutputBorder) + "]" + "\n" + response.Message.Content
		output.SetText(formattedText)
		chatHistory = append(chatHistory, messageData{Role: "assistant", Content: response.Message.Content})
		output.ScrollToEnd()
		app.Draw() // Refresh the UI

		// Save chat history to file
		err = saveChatHistory(chatFile, chatHistory)
		if err != nil {
			log.Fatalf("Error saving chat history: %v", err)
		}
	}

	// MARK: Input Handling
	// Input handling (press Enter to display input in output field)
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			promptText := input.GetText()
			if promptText == "" {
				return
			}

			currentText := output.GetText(false)
			formattedText := currentText + "\n" + "[" + colorToHex(userColor) + "]----------- User input -----------[white]" + "\n" + input.GetText()
			output.SetText(formattedText)
			input.SetText("") // Clear input field

			// Call the model
			go getLLMResponse(promptText)
		}
	})

	// Clear chat history on Ctrl+K
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyF12 {
			err = clearChatHistory(chatFile)
			chatHistory = []messageData{}
			if err != nil {
				log.Fatalf("Error clearing chat history: %v", err)
			}
			return nil
		}
		return event
	})

	// MARK: Run Application ------------------------------------------------------
	// Run the application
	if err := app.SetRoot(grid, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
