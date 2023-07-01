package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sesv2"
)

type FormBody struct {
	Email string `json:"email"`
	HiringOption string `json:"hiringOption"`
	Message string `json:"message"`
	Name string `json:"name"`
	Token string `json:"token"`
}

func passesCaptcha(token string) (result bool) {
	secret := os.Getenv("RECAPTCHA_SECRET")
	if secret == "" || token == "" {
		fmt.Println("secret or token, empty")
		return
	}

	url := fmt.Sprintf("%s?secret=%s&response=%s", "https://www.google.com/recaptcha/api/siteverify",secret, token)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error returned from recaptcha verify url: %v\n", err)
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body %v\n", err)
		return
	}

	type RecaptchaVerifyResponse struct {
		Success bool `json:"success"`
		Score float32 `json:"score"`
		Action string `json:"action"`
		Challenge_ts string `json:"challenge_ts"`
	}

	var rvr RecaptchaVerifyResponse
	err = json.Unmarshal(body, &rvr)
	if err != nil {
		fmt.Printf("Error unmarshalling json %v\n", err)
		return
	}

	fmt.Println(rvr)
	result = rvr.Success
	return
}

func sendEmail(form FormBody) (err error) {
	receiver := os.Getenv("RECEIVER")
	sender := os.Getenv("SENDER")
	sess := session.New()
	svc := sesv2.New(sess)
	subject := fmt.Sprintf("Wagonkered Website Enquiry from %s\n", form.Name)

	_, err = svc.SendEmail(
		&sesv2.SendEmailInput{
			Content: &sesv2.EmailContent{
				Simple: &sesv2.Message{
					Body: &sesv2.Body{
						Text: &sesv2.Content{
							Data: aws.String(fmt.Sprintf("Name: %s\nEmail address: %s\nHiring Option: %s\nMessage:\n%s\n", form.Name, form.Email, form.HiringOption, form.Message)),
						},
					},
					Subject: &sesv2.Content{
						Data: aws.String(subject),
					},
				},
			},
			Destination: &sesv2.Destination{
				ToAddresses: []*string{&sender},
			},
			FromEmailAddress: &receiver,
		},
	)
	return
}

func handler(request events.APIGatewayProxyRequest) (response events.APIGatewayProxyResponse, err error) {
	response.Headers = map[string]string{"Access-Control-Allow-Origin": "https://wagonkered.co.uk"}
	response.StatusCode = 401
	response.Body = "Unable to send email"

	var form FormBody
	json.Unmarshal([]byte(request.Body), &form)

	fmt.Println(form)
	if !passesCaptcha(form.Token) {
		return
	}
	
	err = sendEmail(form)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return
	}
	
	response.StatusCode = 200
	response.Body = "Sucess"
	return
}

func main() {
	lambda.Start(handler)
}
