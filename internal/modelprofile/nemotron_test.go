package modelprofile

import (
	"math"
	"strings"
	"testing"
)

func TestNemotronAdmitsOnlyBaseLanguagesAndBoundedHints(t *testing.T) {
	valid := NemotronOptions{Vocabulary: "Freehand\nNew York", Boost: 3}
	for _, language := range []string{"auto", "en-US", "zh-CN", "uk-UA"} {
		if err := ValidateNemotron(language, valid); err != nil {
			t.Fatal(err)
		}
	}
	for _, language := range []string{"en", "he-IL", "th-TH", "el-GR", "unknown"} {
		if ValidateNemotron(language, valid) == nil {
			t.Fatalf("unqualified language admitted: %s", language)
		}
	}
	for _, options := range []NemotronOptions{{Boost: math.NaN()}, {Boost: 6}, {Vocabulary: strings.Repeat("phrase\n", 33), Boost: 3}, {Vocabulary: "a\x00b", Boost: 3}} {
		if ValidateNemotron("en-US", options) == nil {
			t.Fatal("invalid option admitted")
		}
	}
}

func TestNemotronLanguageTagDoesNotRemoveOrdinaryText(t *testing.T) {
	text, language := StripNemotronLanguageTag("Keep <ordinary> text. <en-US>")
	if text != "Keep <ordinary> text." || language != "en-US" {
		t.Fatal("tag handling changed ordinary text")
	}
}
