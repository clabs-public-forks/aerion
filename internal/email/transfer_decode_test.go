package email

import (
	"encoding/base64"
	"strings"
	"testing"
)

// TestAttachmentsDecodedOnce checks that attachment bodies are decoded only
// by go-message, so content that itself looks encoded survives intact.
func TestAttachmentsDecodedOnce(t *testing.T) {
	const qpContent = "total=41;x =\nnext" // decodes again to "totalA;x next"
	const b64Content = "aGVsbG8gd29ybGQ="  // valid base64 itself
	raw := strings.ReplaceAll(`From: a@example.com
To: b@example.com
Subject: t
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary=XX

--XX
Content-Type: text/plain

body
--XX
Content-Type: text/plain; name=qp.txt
Content-Disposition: attachment; filename=qp.txt
Content-Transfer-Encoding: quoted-printable

total=3D41;x =3D
next
--XX
Content-Type: application/octet-stream; name=b.bin
Content-Disposition: attachment; filename=b.bin
Content-Transfer-Encoding: base64

`+base64.StdEncoding.EncodeToString([]byte(b64Content))+`
--XX--
`, "\n", "\r\n")

	want := map[string]string{"qp.txt": strings.ReplaceAll(qpContent, "\n", "\r\n"), "b.bin": b64Content}
	d := NewAttachmentDownloader(t.TempDir())
	for name, w := range want {
		got, err := d.ExtractAttachmentContent([]byte(raw), name)
		if err != nil {
			t.Fatalf("ExtractAttachmentContent(%s): %v", name, err)
		}
		if string(got) != w {
			t.Errorf("ExtractAttachmentContent(%s) = %q, want %q", name, got, w)
		}
	}

	atts, err := NewAttachmentExtractor().ExtractAttachments("m1", []byte(raw))
	if err != nil {
		t.Fatalf("ExtractAttachments: %v", err)
	}
	for _, a := range atts {
		if w, ok := want[a.Attachment.Filename]; ok && string(a.Content) != w {
			t.Errorf("ExtractAttachments %s = %q, want %q", a.Attachment.Filename, a.Content, w)
		}
	}
	if len(atts) != len(want) {
		t.Errorf("got %d attachments, want %d", len(atts), len(want))
	}
}
