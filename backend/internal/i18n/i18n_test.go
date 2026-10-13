package i18n

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"testing"
)

func request(cookie, acceptLanguage string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
	}
	if acceptLanguage != "" {
		r.Header.Set("Accept-Language", acceptLanguage)
	}
	return r
}

func TestFromRequest(t *testing.T) {
	cases := []struct {
		name, cookie, header, want string
	}{
		{"cookie wins over header", "ru", "de-DE,de;q=0.9", "ru"},
		{"english cookie wins over header", "en", "de", "en"},
		{"polish cookie wins over header", "pl", "de", "pl"},
		{"bogus cookie is ignored", "fr", "de", "de"},
		{"german header", "", "de-DE,de;q=0.9,en;q=0.8", "de"},
		{"russian header", "", "ru", "ru"},
		{"polish header", "", "pl-PL,pl;q=0.9,en;q=0.8", "pl"},
		{"unsupported header", "", "fr-FR", "en"},
		{"nothing", "", "", "en"},
	}
	for _, c := range cases {
		if got := FromRequest(request(c.cookie, c.header)); got != c.want {
			t.Errorf("%s: FromRequest = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTFallsBackToMessage(t *testing.T) {
	const msg = "No catalog has this sentence."
	for _, lang := range []string{"en", "de", "ru", "", "fr"} {
		if got := T(lang, msg); got != msg {
			t.Errorf("T(%q) = %q, want the message unchanged", lang, got)
		}
	}
	if got := T("en", "Invalid request body."); got != "Invalid request body." {
		t.Errorf("T(en) translated: %q", got)
	}
	if got := T("de", "Invalid request body."); got == "Invalid request body." {
		t.Error("T(de) left a catalogued message in English")
	}
}

func TestOfTranslatesAnErrorFromItsFormat(t *testing.T) {
	err := fmt.Errorf("saving: %w", Errorf("extraction_rules must be at most %d characters", 4000))
	if got, want := Of("en", err), "saving: extraction_rules must be at most 4000 characters"; got != want {
		t.Errorf("Of(en) = %q, want the English unchanged: %q", got, want)
	}
	if got, want := Of("de", err), "extraction_rules darf höchstens 4000 Zeichen lang sein"; got != want {
		t.Errorf("Of(de) = %q, want %q", got, want)
	}
	if got, want := T("pl", "Authentication required."), "Wymagane jest uwierzytelnienie."; got != want {
		t.Errorf("T(pl) = %q, want %q", got, want)
	}
	if got, want := Of("pl", err), "extraction_rules może mieć maksymalnie 4000 znaków"; got != want {
		t.Errorf("Of(pl) = %q, want %q", got, want)
	}
	if got := Of("ru", errors.New("Invalid request body.")); got == "Invalid request body." {
		t.Error("Of(ru) left a catalogued plain error in English")
	}
	if got := Of("de", errors.New("not catalogued")); got != "not catalogued" {
		t.Errorf("Of(de) = %q, want an uncatalogued error unchanged", got)
	}
}

var verb = regexp.MustCompile(`%[-+# 0-9.*\[\]]*[a-zA-Z%]`)

func TestCatalogEntriesAreCompleteAndKeepFormatVerbs(t *testing.T) {
	for key, translations := range catalog {
		if len(translations) != 3 {
			t.Errorf("%q: languages %v, want de, ru, and pl", key, translations)
		}
		for lang, value := range translations {
			if value == "" {
				t.Errorf("%q: no %s translation", key, lang)
			}
			if want, got := verb.FindAllString(key, -1), verb.FindAllString(value, -1); !slices.Equal(want, got) {
				t.Errorf("%s %q: verbs %v, want %v", lang, key, got, want)
			}
		}
	}
}
