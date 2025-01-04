package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Question struct {
	ID       int      `json:"id"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Options  []string `json:"options"`
}

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	dbName := os.Getenv("TURSO_DB_URL")
	dbToken := os.Getenv("TURSO_DB_AUTH_TOKEN")

	var err error
	dbString := fmt.Sprintf("%s/?authToken=%s", dbName, dbToken)
	db, err := sql.Open("libsql", dbString)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}
	defer db.Close()

	switch req.HTTPMethod {
	case "GET":
		var question Question
		var optionsJSON string

		err := db.QueryRow("SELECT id, question, answer, options FROM questions ORDER BY RANDOM() LIMIT 1").Scan(&question.ID, &question.Question, &question.Answer, &optionsJSON)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Failed to fetch question"}, nil
		}

		err = json.Unmarshal([]byte(optionsJSON), &question.Options)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Failed to unmarshal options"}, nil
		}

		questionJSON, _ := json.Marshal(question)
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(questionJSON), IsBase64Encoded: false}, nil

	case "POST":
		userAnswer := req.QueryStringParameters["answer"]
		score := 0

		if scoreStr := req.QueryStringParameters["score"]; scoreStr != "" {
			if s, err := fmt.Sscanf(scoreStr, "%d", &score); s != 1 || err != nil {
				score = 0
			}
		}

		questionIDStr := req.QueryStringParameters["question_id"]
		if questionIDStr == "" {
			return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Missing question_id"}, nil
		}

		var correctAnswer string
		err := db.QueryRow("SELECT answer FROM questions WHERE id = ?", questionIDStr).Scan(&correctAnswer)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Failed to fetch correct answer"}, nil
		}

		if strings.TrimSpace(userAnswer) == correctAnswer {
			score = 1
		} else {
			score = 0
		}

		return events.APIGatewayProxyResponse{
			StatusCode:      200,
			Body:            fmt.Sprintf("Your score is: %d", score),
			IsBase64Encoded: false,
		}, nil

	default:
		return events.APIGatewayProxyResponse{StatusCode: 405, Body: "Method Not Allowed"}, nil
	}
}

func main() {
	lambda.Start(handler)
}
