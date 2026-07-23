package signing

import "testing"

func TestSignThenVerify_Accepts(t *testing.T) {
	pub, priv, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	zip := []byte("fake zip bytes")
	manifest := []byte(`{"name":"n"}`)

	sig, err := Sign(priv, zip, manifest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	ok, err := Verify(pub, zip, manifest, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Fatalf("expected signature to verify")
	}
}

func TestVerify_RejectsTamperedZip(t *testing.T) {
	pub, priv, _ := GenerateKey()
	manifest := []byte(`{"name":"n"}`)

	sig, err := Sign(priv, []byte("original zip"), manifest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	ok, err := Verify(pub, []byte("tampered zip"), manifest, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Fatalf("expected signature over different zip bytes to be rejected")
	}
}

func TestVerify_RejectsTamperedManifest(t *testing.T) {
	pub, priv, _ := GenerateKey()
	zip := []byte("some zip")

	sig, err := Sign(priv, zip, []byte(`{"name":"n"}`))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	ok, err := Verify(pub, zip, []byte(`{"name":"tampered"}`), sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Fatalf("expected signature over different manifest bytes to be rejected")
	}
}

func TestVerify_RejectsWrongPublicKey(t *testing.T) {
	_, priv, _ := GenerateKey()
	otherPub, _, _ := GenerateKey()
	zip := []byte("zip")
	manifest := []byte(`{"name":"n"}`)

	sig, err := Sign(priv, zip, manifest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	ok, err := Verify(otherPub, zip, manifest, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Fatalf("expected signature to be rejected under a different publisher's key")
	}
}

func TestVerify_RejectsMalformedSignature(t *testing.T) {
	pub, _, _ := GenerateKey()

	_, err := Verify(pub, []byte("zip"), []byte("manifest"), "not-base64!!!")
	if err == nil {
		t.Fatalf("expected an error for a malformed signature, not a false verification result")
	}
}

func TestEncodeKeyDecodeKey_RoundTrip(t *testing.T) {
	pub, priv, _ := GenerateKey()

	pubStr := EncodePublicKey(pub)
	privStr := EncodePrivateKey(priv)

	gotPub, err := DecodePublicKey(pubStr)
	if err != nil {
		t.Fatalf("DecodePublicKey: %v", err)
	}
	gotPriv, err := DecodePrivateKey(privStr)
	if err != nil {
		t.Fatalf("DecodePrivateKey: %v", err)
	}

	sig, err := Sign(gotPriv, []byte("zip"), []byte("manifest"))
	if err != nil {
		t.Fatalf("Sign with decoded key: %v", err)
	}
	ok, err := Verify(gotPub, []byte("zip"), []byte("manifest"), sig)
	if err != nil || !ok {
		t.Fatalf("round-tripped key pair failed to sign/verify: ok=%v err=%v", ok, err)
	}
}
