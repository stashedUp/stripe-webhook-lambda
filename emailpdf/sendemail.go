package emailpdf

import (
	"fmt"
	"log"
	"net/smtp"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	ss "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/jordan-wright/email"
	"github.com/matcornic/hermes/v2"
)

const (
	PRODUCTEMAIL = "support@warrensbox.com"
	REPNAME      = "Warren from Warrensbox"
	COPYRIGHT    = "Ⓒ 2024 warrensbox.com"
	EMAILSUBJECT = "Attached is your PDF purchase"
	SEND_OK      = "{ \"message\": \"Message sent successfully\"}"
	SEND_NOT_OK  = "{ \"message\": \"Unble to send message\"}"
	IMGHEADER    = "https://kepler-images.s3.us-east-2.amazonaws.com/downloadpdf/downloadpdf-email-logo-350.png"
)

func SendEmail(owner_email string, host string) string {

	fmt.Printf("Attempting to send email...\n")
	session := session.Must(session.NewSession())
	svc := ssm.New(session)

	composedURL := fmt.Sprintf("https://%s.warrensbox.com", host)

	SMTPPASS := getEmailCredential(svc, "SMTP_PASS")

	SMTPUSER := getEmailCredential(svc, "SMTP_USER")

	SMTPEMAIL := getEmailCredential(svc, "SMTP_EMAIL")

	SMTPPORT := getEmailCredential(svc, "SMTP_PORT")

	emailContent := composeEmail()
	emailHeadFoot := composeEmailFooterHeader(composedURL, REPNAME, COPYRIGHT)

	emailBody, errBody := emailHeadFoot.GenerateHTML(emailContent)
	if errBody != nil {
		fmt.Println(errBody)
	}

	bucket := "downloadpdf.org"
	item := getFilename(host, session)

	// 2) Create an AWS session
	sess, _ := ss.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	downloader := s3manager.NewDownloader(sess)
	filepath := "/tmp/" + item
	file, err := os.Create(filepath)
	if err != nil {
		log.Fatalf("Unable to create item %v", err)
	}
	defer file.Close()
	numBytes, err2 := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})

	if err2 != nil {
		log.Fatalf("Unable to download item %q, %v", item, err)
	}

	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")

	e := email.NewEmail()
	e.From = PRODUCTEMAIL
	e.To = []string{owner_email}
	e.Subject = EMAILSUBJECT
	e.HTML = []byte(emailBody)
	e.AttachFile(file.Name())
	auth := smtp.PlainAuth("", SMTPUSER, SMTPPASS, SMTPEMAIL)
	errEmail := e.Send(SMTPPORT, auth)
	if errEmail != nil {
		fmt.Println(errEmail)
		return SEND_NOT_OK
	}
	return SEND_OK
}

func getEmailCredential(svc *ssm.SSM, val string) string {

	param := &ssm.GetParameterInput{
		Name:           aws.String(val),
		WithDecryption: aws.Bool(true),
	}

	paramVal, err := svc.GetParameter(param)
	ErrorExit("GetParameters", err)

	smtpInfo := *paramVal.Parameter.Value
	fmt.Println(smtpInfo)
	return smtpInfo
}

func composeEmail() hermes.Email {

	fmt.Println("Composing email")

	return hermes.Email{
		Body: hermes.Body{
			Title: "Hello",
			Intros: []string{
				"Enjoy your download!",
			},
			//Dictionary: dictionary,
			Outros: []string{
				"Need help, or have questions? Shoot us an email at support@warrensbox.com",
			},
		},
	}
}

func composeEmailFooterHeader(url string, name string, copyright string) hermes.Hermes {

	h := hermes.Hermes{
		Theme: new(hermes.Default),
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: name,
			Link: url,
			// Optional product logo
			Logo:      IMGHEADER,
			Copyright: copyright,
		},
	}

	return h
}

func ErrorExit(msg string, e error) {
	if e != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s, %v\n", msg, e)
		os.Exit(1)
	}
}

func getFilename(host string, session *ss.Session) string {

	svc := dynamodb.New(session)

	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("DownloadPDF"),
		Key: map[string]*dynamodb.AttributeValue{
			"Host": {
				S: aws.String(host),
			},
		},
	})
	if err != nil {
		fmt.Println(err)
	}

	item := Table{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &item)

	if result != nil {
		fmt.Println("Found item:")
		fmt.Println("Source:  ", item.Source)
	}

	return item.Source
}

func GetHost(hostname string) string {

	domain, err := url.Parse(hostname)
	if err != nil {
		fmt.Println("ERROR Parsing URL")
	}

	fmt.Println("DOMAIN", domain.Host)

	hostParts := strings.Split(domain.Host, ".")

	fmt.Println("Host Part", hostParts[0])

	return hostParts[0]
}

type Table struct {
	Source string
}
