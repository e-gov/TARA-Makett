package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
)

const (
	// klientrakenduse hostinimi
	appHost = "localhost"
	// klientrakenduse HTTPS serveri port
	appPort = ":8081"
	// klientrakenduse HTTPS sert.
	appCert = "vault/https.crt"
	// klientrakenduse HTTPS privaatvõti.
	appKey = "vault/https.key"

	// Usaldusankur TARA-Mock-i poole pöördumisel
	rootCAFile = "vault/rootCA.pem"

	// TARA-Mock-i otspunktid
	taraMockAuthorizeEndpoint = "https://localhost:8080/oidc/authorize"
	taraMockTokenEndpoint     = "https://localhost:8080/oidc/token"
	taraMockKeyEndpoint       = "https://localhost:8080/oidc/jwks"

	// OpenID Connect kohane tagasisuunamis-URL
	redirectURI = "https://localhost:8081/return"
)

func main() {

	// Marsruudid
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/", landingPage)
	http.HandleFunc("/login", loginUser)
	http.HandleFunc("/autologin", autologinUser)
	http.HandleFunc("/return", finalize)

	// fileServer serveerib kasutajaliidese muutumatuid faile.
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Käivita HTTPS server
	log.Println("** Klientrakenduse näidis käivitatud pordil 8081 **")
	err := http.ListenAndServeTLS(
		appPort,
		appCert,
		appKey,
		nil)
	if err != nil {
		log.Fatal(err)
	}
}

// LandingPage on klientrakenduse avaleht; kasutaja saab seal sisse logida (/).
func landingPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Valmista ette malli parameetrid.
	type MalliParameetrid struct {
		appHost     string
		appPort     string
		RedirectURI string
	}
	mp := MalliParameetrid{appHost, appPort, redirectURI}

	// Loe avalehe mall, täida ja saada sirvikusse.
	p := filepath.Join("templates", "index.html")
	t, err := template.ParseFiles(p)
	if err != nil {
		fmt.Fprintf(w, "Unable to load template")
		return
	}
	t.Execute(w, mp)
}

// loginUser suunab kasutaja TARA-Mock-i autentima.
func loginUser(w http.ResponseWriter, r *http.Request) {
	// Ümbersuunamis-URL
	ru := taraMockAuthorizeEndpoint + "?" +
		"redirect_uri=" +
		url.PathEscape(redirectURI) + "&" +
		"scope=openid&" +
		"state=1111&" +
		"nonce=2222&" +
		"response_type=code&" +
		"client_id=1"

	fmt.Println("loginUser: Saadan autentimispäringu: ", ru)

	// Suuna kasutaja TARA-Mock-i.
	http.Redirect(w, r, ru, 301)
}

// autologinUser suunab kasutaja TARA-Mock-i automaatautentimisele.
// F-n erib loginUser-st ainult parameetri autologin=<isikukood>
// poolest. TO DO: Kaalu refaktoorimist.
func autologinUser(w http.ResponseWriter, r *http.Request) {
	// Ümbersuunamis-URL
	ru := taraMockAuthorizeEndpoint + "?" +
		"redirect_uri=" +
		url.PathEscape(redirectURI) + "&" +
		"scope=openid&" +
		"state=1111&" +
		"nonce=2222&" +
		"response_type=code&" +
		"client_id=1&" +
		"autologin=36107120334"

	fmt.Println("loginUser: Saadan autentimispäringu: ", ru)

	// Suuna kasutaja TARA-Mock-i.
	http.Redirect(w, r, ru, 301)
}

// finalize : 1) võtab TARA-Moc-st tagasi suunatud kasutaja
// vastu; 2) kutsub välja identsustõendi pärimise; 3) viib sisselogimise
// lõpule - saadab sirvikusse lehe "autenditud". (Otspunkt /client/return).
func finalize(w http.ResponseWriter, r *http.Request) {

	// PassParams koondab lehele "Autenditud" edastatavaid väärtusi.
	type PassParams struct {
		Code        string
		State       string
		Nonce       string
		Isikuandmed string
		Success     bool
	}
	var ps PassParams

	r.ParseForm() // Parsi päringuparameetrid.
	// Võta päringust volituskood, state ja nonce
	ps.Code = getP("code", r)
	ps.State = getP("state", r)
	ps.Nonce = getP("nonce", r)

	// Päri identsustõend
	// t []byte - Identsustõend
	t, ok := getIdentityToken(getP("code", r))
	if !ok {
		fmt.Println("finalize: Identsustõendi pärimine ebaõnnestus")
		ps.Success = false
	} else {
		fmt.Println("finalize: Saadud identsustõend: ", string(t))
		ps.Success = true
	}

	ps.Isikuandmed = t

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Loe lehe "Autenditud" vmall, täida ja saada sirvikusse.
	p := filepath.Join("templates", "autenditud.html")
	tpl, err := template.ParseFiles(p)
	if err != nil {
		fmt.Fprintf(w, "Unable to load template")
		return
	}
	tpl.Execute(w, ps)

}

// healthCheck pakub elutukset (/health).
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	io.WriteString(w, `{"name":"TARA-Mock klientrakendus", "status":"ok"}`)
}
