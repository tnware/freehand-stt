// Package speechlanguage owns language identifiers and S1-mini input classification.
// The picker catalog identifies languages; it does not promise model support.
package speechlanguage

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const Automatic = "auto"

type Option struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// ISO 639-1 names/codes with common speech extensions haw and yue. Names
// follow the ISO 639-2 reference list. This is not a model capability list.
var identifiers = []struct{ code, label string }{
	{"ab", "Abkhazian"},
	{"aa", "Afar"},
	{"af", "Afrikaans"},
	{"ak", "Akan"},
	{"sq", "Albanian"},
	{"am", "Amharic"},
	{"ar", "Arabic"},
	{"an", "Aragonese"},
	{"hy", "Armenian"},
	{"as", "Assamese"},
	{"av", "Avaric"},
	{"ae", "Avestan"},
	{"ay", "Aymara"},
	{"az", "Azerbaijani"},
	{"bm", "Bambara"},
	{"ba", "Bashkir"},
	{"eu", "Basque"},
	{"be", "Belarusian"},
	{"bn", "Bengali"},
	{"bh", "Bihari languages"},
	{"bi", "Bislama"},
	{"nb", "Bokmål, Norwegian"},
	{"bs", "Bosnian"},
	{"br", "Breton"},
	{"bg", "Bulgarian"},
	{"my", "Burmese"},
	{"yue", "Cantonese"},
	{"ca", "Catalan"},
	{"km", "Central Khmer"},
	{"ch", "Chamorro"},
	{"ce", "Chechen"},
	{"ny", "Chichewa"},
	{"zh", "Chinese"},
	{"cu", "Church Slavic"},
	{"cv", "Chuvash"},
	{"kw", "Cornish"},
	{"co", "Corsican"},
	{"cr", "Cree"},
	{"hr", "Croatian"},
	{"cs", "Czech"},
	{"da", "Danish"},
	{"dv", "Divehi"},
	{"nl", "Dutch"},
	{"dz", "Dzongkha"},
	{"en", "English"},
	{"eo", "Esperanto"},
	{"et", "Estonian"},
	{"ee", "Ewe"},
	{"fo", "Faroese"},
	{"fj", "Fijian"},
	{"fi", "Finnish"},
	{"fr", "French"},
	{"ff", "Fulah"},
	{"gd", "Gaelic"},
	{"gl", "Galician"},
	{"lg", "Ganda"},
	{"ka", "Georgian"},
	{"de", "German"},
	{"el", "Greek, Modern (1453-)"},
	{"gn", "Guarani"},
	{"gu", "Gujarati"},
	{"ht", "Haitian"},
	{"ha", "Hausa"},
	{"haw", "Hawaiian"},
	{"he", "Hebrew"},
	{"hz", "Herero"},
	{"hi", "Hindi"},
	{"ho", "Hiri Motu"},
	{"hu", "Hungarian"},
	{"is", "Icelandic"},
	{"io", "Ido"},
	{"ig", "Igbo"},
	{"id", "Indonesian"},
	{"ia", "Interlingua (International Auxiliary Language Association)"},
	{"ie", "Interlingue"},
	{"iu", "Inuktitut"},
	{"ik", "Inupiaq"},
	{"ga", "Irish"},
	{"it", "Italian"},
	{"ja", "Japanese"},
	{"jv", "Javanese"},
	{"kl", "Kalaallisut"},
	{"kn", "Kannada"},
	{"kr", "Kanuri"},
	{"ks", "Kashmiri"},
	{"kk", "Kazakh"},
	{"ki", "Kikuyu"},
	{"rw", "Kinyarwanda"},
	{"ky", "Kirghiz"},
	{"kv", "Komi"},
	{"kg", "Kongo"},
	{"ko", "Korean"},
	{"kj", "Kuanyama"},
	{"ku", "Kurdish"},
	{"lo", "Lao"},
	{"la", "Latin"},
	{"lv", "Latvian"},
	{"li", "Limburgan"},
	{"ln", "Lingala"},
	{"lt", "Lithuanian"},
	{"lu", "Luba-Katanga"},
	{"lb", "Luxembourgish"},
	{"mk", "Macedonian"},
	{"mg", "Malagasy"},
	{"ms", "Malay"},
	{"ml", "Malayalam"},
	{"mt", "Maltese"},
	{"gv", "Manx"},
	{"mi", "Maori"},
	{"mr", "Marathi"},
	{"mh", "Marshallese"},
	{"mn", "Mongolian"},
	{"na", "Nauru"},
	{"nv", "Navajo"},
	{"nd", "Ndebele, North"},
	{"nr", "Ndebele, South"},
	{"ng", "Ndonga"},
	{"ne", "Nepali"},
	{"se", "Northern Sami"},
	{"no", "Norwegian"},
	{"nn", "Norwegian Nynorsk"},
	{"oc", "Occitan (post 1500)"},
	{"oj", "Ojibwa"},
	{"or", "Oriya"},
	{"om", "Oromo"},
	{"os", "Ossetian"},
	{"pi", "Pali"},
	{"pa", "Panjabi"},
	{"fa", "Persian"},
	{"pl", "Polish"},
	{"pt", "Portuguese"},
	{"ps", "Pushto"},
	{"qu", "Quechua"},
	{"ro", "Romanian"},
	{"rm", "Romansh"},
	{"rn", "Rundi"},
	{"ru", "Russian"},
	{"sm", "Samoan"},
	{"sg", "Sango"},
	{"sa", "Sanskrit"},
	{"sc", "Sardinian"},
	{"sr", "Serbian"},
	{"sn", "Shona"},
	{"ii", "Sichuan Yi"},
	{"sd", "Sindhi"},
	{"si", "Sinhala"},
	{"sk", "Slovak"},
	{"sl", "Slovenian"},
	{"so", "Somali"},
	{"st", "Sotho, Southern"},
	{"es", "Spanish"},
	{"su", "Sundanese"},
	{"sw", "Swahili"},
	{"ss", "Swati"},
	{"sv", "Swedish"},
	{"tl", "Tagalog"},
	{"ty", "Tahitian"},
	{"tg", "Tajik"},
	{"ta", "Tamil"},
	{"tt", "Tatar"},
	{"te", "Telugu"},
	{"th", "Thai"},
	{"bo", "Tibetan"},
	{"ti", "Tigrinya"},
	{"to", "Tonga (Tonga Islands)"},
	{"ts", "Tsonga"},
	{"tn", "Tswana"},
	{"tr", "Turkish"},
	{"tk", "Turkmen"},
	{"tw", "Twi"},
	{"ug", "Uighur"},
	{"uk", "Ukrainian"},
	{"ur", "Urdu"},
	{"uz", "Uzbek"},
	{"ve", "Venda"},
	{"vi", "Vietnamese"},
	{"vo", "Volapük"},
	{"wa", "Walloon"},
	{"cy", "Welsh"},
	{"fy", "Western Frisian"},
	{"wo", "Wolof"},
	{"xh", "Xhosa"},
	{"yi", "Yiddish"},
	{"yo", "Yoruba"},
	{"za", "Zhuang"},
	{"zu", "Zulu"},
}

func Options() []Option {
	result := make([]Option, 0, len(identifiers))
	for _, item := range identifiers {
		result = append(result, Option{item.code, item.label})
	}
	return result
}

// Validate permits bounded custom server values and preserves older settings.
// Providers retain responsibility for deciding which selected languages their model supports.
func Validate(value string) error {
	if len(value) > 32 || !utf8.ValidString(value) || strings.ContainsFunc(value, unicode.IsControl) {
		return errors.New("language must be at most 32 UTF-8 bytes with no control characters")
	}
	return nil
}

func Unspecified(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto", "und", "unknown":
		return true
	}
	return false
}

// English recognizes explicit/reported names and language tags. It does not
// inspect transcript text or infer language from a model ID.
func English(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "english" || value == "eng" {
		return true
	}
	base, _, _ := strings.Cut(strings.ReplaceAll(value, "_", "-"), "-")
	return base == "en"
}
