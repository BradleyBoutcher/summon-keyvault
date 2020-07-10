package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/cyberark/summon-keyvault/pkg/summon_keyvault"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/go-autorest/autorest"
	"github.com/karrick/golf"
	log "github.com/sirupsen/logrus"
)
/*
 * You need to set four environment variables before using the app:
 * AZURE_TENANT_ID: Your Azure tenant ID
 * AZURE_CLIENT_ID: Your Azure client ID. This will be an app ID from your AAD.
 * AZURE_CLIENT_SECRET: The secret for the client ID above.
 * KVAULT: The name of your vault (just the name, not the full URL/path)
 *
 * Usage
 * Get the value for a secret in the vault:
 * cmd YOUR_SECRETS_NAME
 *
 */

var (
	vaultName string
)

func main() {
	var help = golf.BoolP('h', "help", false, "show help")
	var version = golf.BoolP('V', "version", false, "show version")
	var verbose = golf.BoolP('v', "verbose", false, "be verbose")
    var setDebug  = golf.BoolP('d', "debug", false, "use debug")

	golf.Parse()
	args := golf.Args()

	if *version {
		fmt.Println(summon_keyvault.VERSION)
		os.Exit(0)
	}
	if *help {
		golf.Usage()
		os.Exit(0)
	}
	if len(args) == 0 {
		golf.Usage()
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true, DisableLevelTruncation: true})
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	PassManager(args[0], setDebug)
}

func PassManager(secret string, setDebug *bool) {
	if os.Getenv("AZURE_TENANT_ID") == "" || os.Getenv("AZURE_CLIENT_ID") == "" ||
		os.Getenv("AZURE_CLIENT_SECRET") == "" || os.Getenv("KVAULT") == "" {
		fmt.Println("env vars not set, exiting...")
		os.Exit(1)
	}
	vaultName = os.Getenv("KVAULT")

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		fmt.Printf("unable to create vault authorizer: %v\n", err)
		os.Exit(1)
	}

	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	if *setDebug {
		basicClient.RequestInspector = logRequest()
		basicClient.ResponseInspector = logResponse()
	}

	if flag.NArg() == 1 && flag.NFlag() <= 0 {
		getSecret(basicClient, secret)
	}
}

func getSecret(basicClient keyvault.BaseClient, secretName string) {
	secretResp, err := basicClient.GetSecret(
		context.Background(),
		fmt.Sprintf("https://%s.vault.azure.net", vaultName),
		secretName,
		"")

	if err != nil {
		fmt.Printf("unable to get value for secret: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(*secretResp.Value)
}

func logRequest() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				log.Println(err)
			}
			dump, _ := httputil.DumpRequestOut(r, true)
			log.Println(string(dump))
			return r, err
		})
	}
}

func logResponse() autorest.RespondDecorator {
	return func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			err := p.Respond(r)
			if err != nil {
				log.Println(err)
			}
			dump, _ := httputil.DumpResponse(r, true)
			log.Println(string(dump))
			return err
		})
	}
}
