package shared

import (
	"regexp"
	"strings"

	"golang.org/x/text/language"
)

// maxAcceptLanguageLength borne le travail fait sur un en-tête hostile : une
// valeur plus longue, qu'aucun client réel n'envoie, donne la langue par défaut.
const maxAcceptLanguageLength = 1024

var frenchBase, _ = language.ParseBase("fr")

// acceptLanguageItem est un élément de l'en-tête au sens strict de la RFC 9110
// (plage de langue séparée par des tirets, q entre 0 et 1 avec trois décimales
// au plus). x/text accepte davantage (q supérieur à 1, tirets bas) : un élément
// qui sort de cette forme rend tout l'en-tête invalide.
var acceptLanguageItem = regexp.MustCompile(`^(?:[A-Za-z]{1,8}(?:-[A-Za-z0-9]{1,8})*|\*)(?:\s*;\s*[qQ]=(?:0(?:\.[0-9]{0,3})?|1(?:\.0{0,3})?))?$`)

// NegotiateLanguage choisit la langue des messages d'après les lignes
// Accept-Language de la requête : le français si la langue prise en charge la
// plus demandée (préférence q la plus haute, la première à égalité) est une
// variante de fr, l'anglais dans tous les autres cas, y compris un en-tête
// absent ou invalide (RFC 9110, section 12.5.4).
func NegotiateLanguage(headerLines []string) Lang {
	header := strings.Join(headerLines, ",")
	if header == "" || len(header) > maxAcceptLanguageLength {
		return LangEnglish
	}
	for _, item := range strings.Split(header, ",") {
		if item = strings.TrimSpace(item); item != "" && !acceptLanguageItem.MatchString(item) {
			return LangEnglish
		}
	}
	tags, weights, err := language.ParseAcceptLanguage(header)
	if err != nil {
		return LangEnglish
	}

	best, bestWeight := LangEnglish, float32(0)
	for i, tag := range tags {
		base, _ := tag.Base()
		var lang Lang
		switch base {
		case frenchBase:
			lang = LangFrench
		default:
			if englishBase, _ := language.English.Base(); base == englishBase {
				lang = LangEnglish
			} else {
				continue
			}
		}
		if weights[i] > bestWeight {
			best, bestWeight = lang, weights[i]
		}
	}
	return best
}
