package zerotrue_test

import (
	"context"
	"fmt"
	"log"

	zerotrue "github.com/zerotrue/sdk-go"
)

func ExampleNewClient() {
	client, err := zerotrue.NewClient("zt_your_api_key_here",
		zerotrue.WithBaseURL("https://api.zerotrue.com"),
		zerotrue.WithMaxRetries(3),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
	fmt.Println("Client created")
	// Output: Client created
}

func ExampleClient_AnalyzeText() {
	client, err := zerotrue.NewClient("zt_your_api_key_here",
		zerotrue.WithBaseURL("https://api.zerotrue.com"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Analyze text for AI-generated content
	result, err := client.AnalyzeText(context.Background(), "Some text to analyze", &zerotrue.AnalyzeOptions{
		IsDeepScan:    false,
		IsPrivateScan: true,
	})
	if err != nil {
		// Handle specific error types
		log.Fatal(err)
	}

	_ = result
	// result.AIProbability, result.HumanProbability, etc.
}

func ExampleClient_AnalyzeFile() {
	client, err := zerotrue.NewClient("zt_your_api_key_here",
		zerotrue.WithBaseURL("https://api.zerotrue.com"),
	)
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.AnalyzeFile(context.Background(), "image.png", nil)
	if err != nil {
		log.Fatal(err)
	}

	_ = result
}

func ExampleClient_CreateCheck() {
	client, err := zerotrue.NewClient("zt_your_api_key_here",
		zerotrue.WithBaseURL("https://api.zerotrue.com"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Async flow: create check
	check, err := client.CreateCheck(context.Background(), zerotrue.CheckInput{
		Type:  "text",
		Value: "Check this text",
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	_ = check
	// check.ID, check.Status == "queued"
	// Then use client.GetCheck(ctx, check.ID) or client.WaitForResult(ctx, check.ID)
}
