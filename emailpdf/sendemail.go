package emailpdf

import (
	"fmt"
	"log"
	"net/smtp"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	ss "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/jordan-wright/email"
	"github.com/matcornic/hermes/v2"
)

const (
	PRODUCTEMAIL = "support@warrensbox.com"
	PRODUCTURL   = "https://holy-bible.download.org"
	REPNAME      = "Nathan from DownloadPDF.org"
	COPYRIGHT    = "Ⓒ 2021 DownloadPDF.org"
	EMAILSUBJECT = "Attached is your PDF purchase"
	SEND_OK      = "{ \"message\": \"Message sent successfully\"}"
	SEND_NOT_OK  = "{ \"message\": \"Unble to send message\"}"
	IMGHEADER    = "https://kepler-images.s3.us-east-2.amazonaws.com/downloadpdf/downloadpdf-email-logo-200.png"
)

func SendEmail(owner_email string, msg_content string) string {

	fmt.Printf("Attempting to send email...\n")
	session := session.Must(session.NewSession())
	svc := ssm.New(session)

	SMTPPASS := getEmailCredential(svc, "SMTP_PASS")

	SMTPUSER := getEmailCredential(svc, "SMTP_USER")

	SMTPEMAIL := getEmailCredential(svc, "SMTP_EMAIL")

	SMTPPORT := getEmailCredential(svc, "SMTP_PORT")

	emailContent := composeEmail(msg_content)
	emailHeadFoot := composeEmailFooterHeader(PRODUCTURL, REPNAME, COPYRIGHT)

	emailBody, errBody := emailHeadFoot.GenerateHTML(emailContent)
	if errBody != nil {
		fmt.Println(errBody)
	}

	bucket := "downloadpdf.org"
	item := "bb.pdf"

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

func composeEmail(msg_content string) hermes.Email {

	fmt.Println("Composing email")

	// dictionary := []hermes.Entry{
	// 	{Key: "Message", Value: msg_content},
	// }
	// }

	return hermes.Email{
		Body: hermes.Body{
			Title: "Hello",
			Intros: []string{
				"Enjoy your download!",
			},
			//Dictionary: dictionary,
			Outros: []string{
				"Need help, or have questions? Shoot us an email at support@warrensbox.com.",
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
