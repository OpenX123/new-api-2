package common

import (
	"strings"
	"testing"
)

func TestBuildMailWithAttachment(t *testing.T) {
	SystemName = "TestSys"
	SMTPFrom = "noreply@example.com"
	mail, err := BuildMailWithAttachment("发票", "user@example.com",
		"<p>请查收</p>", "invoice.pdf", "application/pdf", []byte("%PDF-1.7"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(mail)
	for _, want := range []string{
		"Content-Type: multipart/mixed",
		"text/html; charset=UTF-8",
		`filename="invoice.pdf"`,
		"Content-Transfer-Encoding: base64",
		"JVBERi0xLjc=", // base64("%PDF-1.7")
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("mail missing %q\n%s", want, s)
		}
	}

	// 注入防护：filename 中含引号与换行字符，应被清除
	mail2, err := BuildMailWithAttachment("test", "user@example.com",
		"<p>test</p>", "bad\"name\r\n.pdf", "application/pdf", []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	s2 := string(mail2)
	if !strings.Contains(s2, `filename="badname.pdf"`) {
		t.Fatalf("expected sanitized filename in mail, got:\n%s", s2)
	}
	if strings.Contains(s2, `bad"name`) {
		t.Fatalf("unsanitized filename still present in mail:\n%s", s2)
	}
}
