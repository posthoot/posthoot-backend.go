package utils

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"
)

// EmailAttachment represents a single attachment in an email.
type EmailAttachment struct {
	Filename string // The original filename of the attachment.
	Data     []byte // The raw byte data of the attachment.
	MIMEType string // The MIME type of the attachment (e.g., "application/pdf").
}

// ParsedMail represents the structured data extracted from an email.
type ParsedMail struct {
	BodyText    string            // Plain text version of the email body.
	BodyHTML    string            // HTML version of the email body.
	ReplyTo     []*mail.Address   // List of Reply-To addresses.
	Cc          []*mail.Address   // List of CC'd addresses.
	To          []*mail.Address   // List of To addresses.
	Bcc         []*mail.Address   // List of BCC'd addresses (usually not available in received mail).
	From        []*mail.Address   // List of From addresses.
	Subject     string            // The subject of the email.
	Date        time.Time         // The date the email was sent.
	MessageID   string            // The unique Message-ID of the email.
	Attachments []EmailAttachment // A slice of attachments found in the email.
}

// ParseEmail takes an io.Reader containing raw email data and parses it
// into a ParsedMail struct.
func ParseEmail(emailReader io.Reader) (*ParsedMail, error) {
	parsedMail := &ParsedMail{} // Use a pointer to modify it

	msg, err := mail.ReadMessage(emailReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read email message: %w", err)
	}

	// Parse basic headers
	parsedMail.Subject = msg.Header.Get("Subject")
	parsedMail.MessageID = msg.Header.Get("Message-ID")

	dateStr := msg.Header.Get("Date")
	if dateStr != "" {
		parsedMail.Date, err = mail.ParseDate(dateStr)
		if err != nil {
			// Log non-fatal error, or decide if this should be fatal
			log.Printf("Warning: Failed to parse date '%s': %v", dateStr, err)
		}
	}

	fromStr := msg.Header.Get("From")
	if fromStr != "" {
		parsedMail.From, err = mail.ParseAddressList(fromStr)
		if err != nil {
			log.Printf("Warning: Failed to parse 'From' addresses '%s': %v", fromStr, err)
		}
	}

	toStr := msg.Header.Get("To")
	if toStr != "" {
		parsedMail.To, err = mail.ParseAddressList(toStr)
		if err != nil {
			log.Printf("Warning: Failed to parse 'To' addresses '%s': %v", toStr, err)
		}
	}

	ccStr := msg.Header.Get("Cc")
	if ccStr != "" {
		parsedMail.Cc, err = mail.ParseAddressList(ccStr)
		if err != nil {
			log.Printf("Warning: Failed to parse 'Cc' addresses '%s': %v", ccStr, err)
		}
	}

	replyToStr := msg.Header.Get("Reply-To")
	if replyToStr != "" {
		parsedMail.ReplyTo, err = mail.ParseAddressList(replyToStr)
		if err != nil {
			log.Printf("Warning: Failed to parse 'Reply-To' addresses '%s': %v", replyToStr, err)
		}
	}
	// Note: BCC is typically not present in received email headers.

	// Process the body and attachments
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Content-Type: %w", err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return nil, fmt.Errorf("multipart email but no boundary found")
		}
		err = parseMultipartBody(msg.Body, boundary, parsedMail)
		if err != nil {
			return nil, fmt.Errorf("failed to parse multipart body: %w", err)
		}
	} else {
		// Not a multipart email, body is simple
		bodyBytes, readErr := io.ReadAll(msg.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read body of non-multipart email: %w", readErr)
		}
		// Determine if it's HTML or plain text based on Content-Type
		if mediaType == "text/html" {
			parsedMail.BodyHTML = string(bodyBytes)
		} else { // Default to plain text
			parsedMail.BodyText = string(bodyBytes)
		}
	}

	return parsedMail, nil
}

// parseMultipartBody handles the parsing of multipart email sections.
// It modifies the passed ParsedMail struct directly.
func parseMultipartBody(body io.Reader, boundary string, mailData *ParsedMail) error {
	mr := multipart.NewReader(body, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break // End of parts
		}
		if err != nil {
			return fmt.Errorf("error reading next multipart section: %w", err)
		}
		defer part.Close() // Ensure part is closed

		partMediaType, partParams, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			log.Printf("Warning: Error parsing part Content-Type: %v. Skipping part.", err)
			continue
		}

		// Check for attachments
		if cd := part.Header.Get("Content-Disposition"); cd != "" && (strings.HasPrefix(strings.ToLower(cd), "attachment") || strings.HasPrefix(strings.ToLower(cd), "inline")) {
			filename := part.FileName()
			// If filename is empty, try to get it from Content-Disposition parameters
			// This can happen with some mail clients for inline attachments
			if filename == "" {
				_, cdParams, cdErr := mime.ParseMediaType(cd)
				if cdErr == nil && cdParams["filename"] != "" {
					filename = cdParams["filename"]
				} else {
					// If still no filename, generate a placeholder or skip
					log.Printf("Warning: Attachment found with no filename. Content-Type: %s", partMediaType)
					// filename = "untitled_attachment" // Or skip this attachment
				}
			}

			var attachmentData []byte
			var readErr error
			encoding := part.Header.Get("Content-Transfer-Encoding")

			var partReader io.Reader = part
			if strings.ToLower(encoding) == "base64" {
				partReader = base64.NewDecoder(base64.StdEncoding, part)
			}
			// Add handling for "quoted-printable" if necessary
			// else if strings.ToLower(encoding) == "quoted-printable" {
			// 	partReader = quotedprintable.NewReader(part)
			// }

			attachmentData, readErr = io.ReadAll(partReader)
			if readErr != nil {
				log.Printf("Warning: Error reading attachment data for '%s': %v. Skipping attachment.", filename, readErr)
				continue
			}

			mailData.Attachments = append(mailData.Attachments, EmailAttachment{
				Filename: filename,
				Data:     attachmentData,
				MIMEType: partMediaType,
			})
		} else if strings.HasPrefix(partMediaType, "multipart/alternative") {
			// This part itself contains alternative versions (e.g., text and HTML)
			altBoundary := partParams["boundary"]
			if altBoundary == "" {
				log.Println("Warning: Multipart/alternative part found but no boundary for its inner parts. Skipping.")
				continue
			}
			// Recursively parse the alternative parts.
			// Note: This simple recursion doesn't accumulate attachments from deeper levels,
			// but for typical email structures (alternative for body, then mixed for attachments), it's okay.
			// A more robust parser might pass down the main `mailData` to aggregate all parts.
			// For now, we assume attachments are at the top multipart/mixed level or directly specified.
			err := parseMultipartAlternative(part, altBoundary, mailData)
			if err != nil {
				log.Printf("Warning: Could not parse multipart/alternative: %v", err)
			}
		} else if partMediaType == "text/plain" {
			// Only set if not already set (preferring one from multipart/alternative if present)
			if mailData.BodyText == "" {
				bodyBytes, readErr := io.ReadAll(part)
				if readErr != nil {
					log.Printf("Warning: Error reading plain text body part: %v", readErr)
					continue
				}
				mailData.BodyText = string(bodyBytes)
			}
		} else if partMediaType == "text/html" {
			// Only set if not already set
			if mailData.BodyHTML == "" {
				bodyBytes, readErr := io.ReadAll(part)
				if readErr != nil {
					log.Printf("Warning: Error reading HTML body part: %v", readErr)
					continue
				}
				mailData.BodyHTML = string(bodyBytes)
			}
		}
	}
	return nil
}

// parseMultipartAlternative handles the parsing of multipart/alternative sections.
func parseMultipartAlternative(alternativeBody io.Reader, boundary string, mailData *ParsedMail) error {
	mr := multipart.NewReader(alternativeBody, boundary)
	for {
		altPart, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading next alternative part: %w", err)
		}
		defer altPart.Close()

		altPartMediaType, _, err := mime.ParseMediaType(altPart.Header.Get("Content-Type"))
		if err != nil {
			log.Printf("Warning: Error parsing alternative part Content-Type: %v. Skipping part.", err)
			continue
		}

		bodyBytes, readErr := io.ReadAll(altPart)
		if readErr != nil {
			log.Printf("Warning: Error reading data from alternative part (%s): %v", altPartMediaType, readErr)
			continue
		}

		if altPartMediaType == "text/plain" {
			mailData.BodyText = string(bodyBytes)
		} else if altPartMediaType == "text/html" {
			mailData.BodyHTML = string(bodyBytes)
		}
	}
	return nil
}

// Helper function to format addresses for printing (optional)
func FormatAddresses(addrs []*mail.Address) string {
	if len(addrs) == 0 {
		return "N/A"
	}
	var out []string
	for _, a := range addrs {
		out = append(out, a.String())
	}
	return strings.Join(out, ", ")
}

// Helper function to truncate strings for printing (optional)
func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
