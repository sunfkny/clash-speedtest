package ip

import (
	"testing"
	"time"
)

func TestGenerateNodeName(t *testing.T) {
	nameCount := make(map[string]int)

	// Test first occurrence - no suffix
	name1 := GenerateNodeName("US", 0, 10*1024*1024, 0, nameCount)
	expected1 := "🇺🇸 US 001 | ⬇️ 10.00MB/s"
	if name1 != expected1 {
		t.Errorf("Expected %s, got %s", expected1, name1)
	}

	// Test second occurrence - should have -01 suffix
	name2 := GenerateNodeName("US", 0, 10*1024*1024, 0, nameCount)
	expected2 := "🇺🇸 US 002 | ⬇️ 10.00MB/s"
	if name2 != expected2 {
		t.Errorf("Expected %s, got %s", expected2, name2)
	}

	// Test third occurrence - should have -02 suffix
	name3 := GenerateNodeName("US", 0, 10*1024*1024, 0, nameCount)
	expected3 := "🇺🇸 US 003 | ⬇️ 10.00MB/s"
	if name3 != expected3 {
		t.Errorf("Expected %s, got %s", expected3, name3)
	}

	// Test different country - no suffix
	name4 := GenerateNodeName("HK", 0, 5*1024*1024, 0, nameCount)
	expected4 := "🇭🇰 HK 001 | ⬇️ 5.00MB/s"
	if name4 != expected4 {
		t.Errorf("Expected %s, got %s", expected4, name4)
	}

	// Test different speed - no suffix
	name5 := GenerateNodeName("US", 0, 15*1024*1024, 0, nameCount)
	expected5 := "🇺🇸 US 004 | ⬇️ 15.00MB/s"
	if name5 != expected5 {
		t.Errorf("Expected %s, got %s", expected5, name5)
	}
}

func TestGenerateNodeNameUploadFallback(t *testing.T) {
	nameCount := make(map[string]int)

	name := GenerateNodeName("JP", 0, 0, 8*1024*1024, nameCount)
	expected := "🇯🇵 JP 001 | ⬆️ 8.00MB/s"
	if name != expected {
		t.Errorf("Expected %s, got %s", expected, name)
	}
}

func TestGenerateNodeNameUnknownCountry(t *testing.T) {
	nameCount := make(map[string]int)

	name := GenerateNodeName("XX", 0, 10*1024*1024, 0, nameCount)
	expected := "🏳️ XX 001 | ⬇️ 10.00MB/s"
	if name != expected {
		t.Errorf("Expected %s, got %s", expected, name)
	}
}

func TestGenerateNodeNameFromTemplate(t *testing.T) {
	nameCount := make(map[string]int)

	// custom template
	name, err := GenerateNodeNameFromTemplate("{{.CountryCode}}-{{.Index}} {{.Speed}}MB/s", "US", 0, 10*1024*1024, 0, nameCount)
	if err != nil {
		t.Fatalf("template error: %v", err)
	}
	expected := "US-001 10.00MB/s"
	if name != expected {
		t.Errorf("Expected %s, got %s", expected, name)
	}

	// empty template uses default
	nameCount2 := make(map[string]int)
	name2, err := GenerateNodeNameFromTemplate("", "HK", 0, 5*1024*1024, 0, nameCount2)
	if err != nil {
		t.Fatalf("template error: %v", err)
	}
	expected2 := "🇭🇰 HK 001 | ⬇️ 5.00MB/s"
	if name2 != expected2 {
		t.Errorf("Expected %s, got %s", expected2, name2)
	}

	// invalid template returns error
	_, err = GenerateNodeNameFromTemplate("{{.Invalid", "US", 0, 0, 0, make(map[string]int))
	if err == nil {
		t.Error("expected parse error for invalid template")
	}
}

func TestGenerateNodeNameTTFBModeWithDefaultTemplate(t *testing.T) {
	nameCount := make(map[string]int)

	name := GenerateNodeName("SG", 120*time.Millisecond, 0, 0, nameCount)
	expected := "🇸🇬 SG 001 | ⚡ 120.00ms"
	if name != expected {
		t.Errorf("Expected %s, got %s", expected, name)
	}
}

func TestGenerateNodeNameFromTemplateTTFBModeLatencyField(t *testing.T) {
	nameCount := make(map[string]int)

	name, err := GenerateNodeNameFromTemplate("{{.CountryCode}}-{{.Index}} {{.LatencyMs}}ms", "DE", 86*time.Millisecond, 0, 0, nameCount)
	if err != nil {
		t.Fatalf("template error: %v", err)
	}
	expected := "DE-001 86ms"
	if name != expected {
		t.Errorf("Expected %s, got %s", expected, name)
	}
}
