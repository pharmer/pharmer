package cloud

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"

	"github.com/appscode/go/log"
	term "github.com/appscode/go/term"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	goauth2 "golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	crmgr "google.golang.org/api/cloudresourcemanager/v1"
	iam "google.golang.org/api/iam/v1"
)

// https://developers.google.com/identity/protocols/OAuth2InstalledApp
const (
	googleOauth2ClientID     = "37154062056-220683ek37naab43v23vc5qg01k1j14g.apps.googleusercontent.com"
	googleOauth2ClientSecret = "pB9ITCuMPLj-bkObrTqKbt57"
)

var gauthConfig goauth2.Config

func IssueGCECredential(name string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Infoln("Oauth2 callback receiver listening on", listener.Addr())

	gauthConfig = goauth2.Config{
		Endpoint:     google.Endpoint,
		ClientID:     googleOauth2ClientID,
		ClientSecret: googleOauth2ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/cloudplatformprojects.readonly", "https://www.googleapis.com/auth/iam"},
		RedirectURL:  "http://" + listener.Addr().String(),
	}
	// PromptSelectAccount allows a user who has multiple accounts at the authorization server
	// to select amongst the multiple accounts that they may have current sessions for.
	// eg: https://developers.google.com/identity/protocols/OpenIDConnect
	promptSelectAccount := oauth2.SetAuthURLParam("prompt", "select_account")
	codeURL := gauthConfig.AuthCodeURL("/", promptSelectAccount)

	log.Infoln("Auhtorization code URL:", codeURL)
	open.Start(codeURL)

	http.HandleFunc("/", handleGoogleAuth)
	return http.Serve(listener, nil)
}

// https://developers.google.com/identity/protocols/OAuth2InstalledApp
func handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		return
	}
	token, err := gauthConfig.Exchange(context.Background(), code)
	term.ExitOnError(err)

	client := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token.AccessToken,
	}))

	rmgrClient, err := crmgr.New(client)
	term.ExitOnError(err)

	// Enable API: https://console.developers.google.com/apis/api/cloudresourcemanager.googleapis.com/overview?project=tigerworks-kube
	presp, err := rmgrClient.Projects.List().Do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(presp.Projects) == 0 {
		http.Error(w, "No Google cloud project exists. Please create new Google Cloud project from web console: https://console.cloud.google.com", http.StatusInternalServerError)
		return
	}

	projects := make([]string, len(presp.Projects))
	for i, p := range presp.Projects {
		projects[i] = p.Name
	}
	_, project := term.List(projects)

	iamClient, err := iam.New(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	saName := "appctl"
	saFQN := fmt.Sprintf("projects/%v/serviceAccounts/%v@%v.iam.gserviceaccount.com", project, saName, project)
	_, err = iamClient.Projects.ServiceAccounts.Get(saFQN).Do()
	if err != nil {
		_, err = iamClient.Projects.ServiceAccounts.Create("projects/"+project, &iam.CreateServiceAccountRequest{
			AccountId: saName,
			ServiceAccount: &iam.ServiceAccount{
				DisplayName: saName,
			},
		}).Do()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	sKey, err := iamClient.Projects.ServiceAccounts.Keys.Create(saFQN, &iam.CreateServiceAccountKeyRequest{}).Do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := base64.StdEncoding.DecodeString(sKey.PrivateKeyData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
