package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/stashedup/stripe-webhook-lambda/emailpdf"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/webhook"
)

var (
	HTTPMethodNotSupported = errors.New("no name was provided in the HTTP body")
)

// ErrorResponseMessage represents the structure of the error
// object sent in failed responses.
type ErrorResponseMessage struct {
	Message string `json:"message"`
}

// ErrorResponse represents the structure of the error object sent
// in failed responses.
type ErrorResponse struct {
	Error *ErrorResponseMessage `json:"error"`
}

//test comment
const (
	DEFAULT = "http://example.com"
)

var (
	statusCode int
)

func init() {

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// For sample support and debugging, not required for production:
	stripe.SetAppInfo(&stripe.AppInfo{
		Name:    "stripe-samples/checkout-one-time-payments",
		Version: "0.0.1",
		URL:     "https://github.com/stripe-samples/checkout-one-time-payments",
	})
}

//HandleRequest incoming request
func HandleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	redirect := make(map[string]string)
	statusCode = 200

	redirect["Location"] = DEFAULT
	redirect["Access-Control-Allow-Origin"] = "*"
	redirect["Access-Control-Allow-Headers"] = "*"
	fmt.Println(request.HTTPMethod)
	fmt.Println("request.HTTPMethod")
	if request.HTTPMethod == "GET" {
		fmt.Printf("GET METHOD\n")
		statusCode = 200
		return events.APIGatewayProxyResponse{Headers: redirect, StatusCode: statusCode}, nil
	} else if request.HTTPMethod == "POST" {

		body := handleWebhook(&request)

		return events.APIGatewayProxyResponse{Headers: redirect, StatusCode: statusCode, Body: body}, nil

	} else {
		fmt.Printf("NEITHER\n")
		return events.APIGatewayProxyResponse{}, HTTPMethodNotSupported
	}

}

func main() {
	lambda.Start(HandleRequest)
}

func writeJSON(v interface{}) string {
	fmt.Println("Attempting to write to JSON")

	var buf bytes.Buffer
	var res bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		log.Printf("json.NewEncoder.Encode: %v", err)
		return string(res.Bytes())
	}

	if _, err := io.Copy(&res, &buf); err != nil {
		log.Printf("io.Copy: %v", err)
		return string(res.Bytes())
	}
	return string(res.Bytes())

}

func handleWebhook(r *events.APIGatewayProxyRequest) string {

	event, err := webhook.ConstructEvent([]byte(r.Body), r.Headers["Stripe-Signature"], os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		log.Printf("webhook.ConstructEvent: %v", err)
		return ""
	}
	resp := RespHook{}

	json.Unmarshal([]byte(r.Body), &resp)

	if event.Type == "checkout.session.completed" {
		fmt.Println("Checkout Session completed!")
		fmt.Println(resp.Data.Object.CustomerEmail)
		fmt.Println(resp.Data.Object.AmountTotal)

		custEmail := resp.Data.Object.CustomerEmail
		host := emailpdf.GetHost(resp.Data.Object.CancelURL)
		emailpdf.SendEmail(custEmail, host)
	}

	return writeJSON(struct {
		SessionID string `json:"sessionId"`
	}{
		SessionID: "s.ID",
	})
}

type RespHook struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	APIVersion string `json:"api_version"`
	Created    int    `json:"created"`
	Data       struct {
		Object struct {
			ID                  string      `json:"id"`
			Object              string      `json:"object"`
			AllowPromotionCodes interface{} `json:"allow_promotion_codes"`
			AmountSubtotal      int         `json:"amount_subtotal"`
			AmountTotal         int         `json:"amount_total"`
			AutomaticTax        struct {
				Enabled bool        `json:"enabled"`
				Status  interface{} `json:"status"`
			} `json:"automatic_tax"`
			BillingAddressCollection interface{} `json:"billing_address_collection"`
			CancelURL                string      `json:"cancel_url"`
			ClientReferenceID        interface{} `json:"client_reference_id"`
			Currency                 string      `json:"currency"`
			Customer                 string      `json:"customer"`
			CustomerDetails          struct {
				Email     string        `json:"email"`
				TaxExempt string        `json:"tax_exempt"`
				TaxIds    []interface{} `json:"tax_ids"`
			} `json:"customer_details"`
			CustomerEmail string      `json:"customer_email"`
			Livemode      bool        `json:"livemode"`
			Locale        interface{} `json:"locale"`
			Metadata      struct {
			} `json:"metadata"`
			Mode                 string `json:"mode"`
			PaymentIntent        string `json:"payment_intent"`
			PaymentMethodOptions struct {
			} `json:"payment_method_options"`
			PaymentMethodTypes        []string    `json:"payment_method_types"`
			PaymentStatus             string      `json:"payment_status"`
			SetupIntent               interface{} `json:"setup_intent"`
			Shipping                  interface{} `json:"shipping"`
			ShippingAddressCollection interface{} `json:"shipping_address_collection"`
			SubmitType                interface{} `json:"submit_type"`
			Subscription              interface{} `json:"subscription"`
			SuccessURL                string      `json:"success_url"`
			TotalDetails              struct {
				AmountDiscount int `json:"amount_discount"`
				AmountShipping int `json:"amount_shipping"`
				AmountTax      int `json:"amount_tax"`
			} `json:"total_details"`
			URL interface{} `json:"url"`
		} `json:"object"`
	} `json:"data"`
	Livemode        bool `json:"livemode"`
	PendingWebhooks int  `json:"pending_webhooks"`
	Request         struct {
		ID             interface{} `json:"id"`
		IdempotencyKey interface{} `json:"idempotency_key"`
	} `json:"request"`
	Type string `json:"type"`
}
