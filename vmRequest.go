package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type (
	vraVars struct {
		username      string
		baseURI       string
		password      string
		tenant        string
		businessGroup string
		blueprint     string
	}
	AuthData struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Tenant   string `json:"tenant"`
	}
)

var VraVars vraVars

func setVars() {
	if len(os.Args) != 1 {
		VraVars.blueprint = os.Args[1]
	} else {
		fmt.Println("blueprint name required as argument")
		os.Exit(1)
	}

	missingVars := []string{}
	if os.Getenv("VRA_ENDPOINT") != "" {
		VraVars.baseURI = os.Getenv("VRA_ENDPOINT")
	} else {
		missingVars = append(missingVars, "VRA_ENDPOINT")
	}
	if os.Getenv("VRA_USER") != "" {
		VraVars.username = os.Getenv("VRA_USER")
	} else {
		missingVars = append(missingVars, "VRA_USER")
	}
	if os.Getenv("VRA_PASSWORD") != "" {
		VraVars.password = os.Getenv("VRA_PASSWORD")
	} else {
		missingVars = append(missingVars, "VRA_PASSWORD")
	}
	if os.Getenv("VRA_TENANT") != "" {
		VraVars.tenant = os.Getenv("VRA_TENANT")
	} else {
		missingVars = append(missingVars, "VRA_TENANT")
	}
	if os.Getenv("VRA_BG") != "" {
		VraVars.businessGroup = os.Getenv("VRA_BG")
	} else {
		missingVars = append(missingVars, "VRA_BG")
	}

	if len(missingVars) != 0 {
		fmt.Println("Missing the following environment variables:")
		fmt.Println(missingVars)
		os.Exit(1)
	}
}

func getToken(client *http.Client) string {
	authURL := VraVars.baseURI + "/identity/api/tokens"
	postData := &AuthData{VraVars.username, VraVars.password, VraVars.tenant}
	b, _ := json.Marshal(postData)
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var d interface{}
	if err := json.Unmarshal(body, &d); err != nil {
		panic(err)
	}
	iface := d.(map[string]interface{})
	return iface["id"].(string)
}

func getTemplate(client *http.Client, auth string, catalogID string, businessGroupID string) []byte {
	templateURL := fmt.Sprintf("%s/catalog-service/api/consumer/entitledCatalogItems/%s/requests/template", VraVars.baseURI, catalogID)
	req, _ := http.NewRequest("GET", templateURL, nil)
	req.Header.Add("Authorization", auth)
	q := req.URL.Query()
	q.Add("businessGroupId", businessGroupID)
	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return body
}

func invokeTemplate(client *http.Client, auth string, template []byte, catalogID string) string {
	templateURL := fmt.Sprintf("%s/catalog-service/api/consumer/entitledCatalogItems/%s/requests", VraVars.baseURI, catalogID)
	req, err := http.NewRequest("POST", templateURL, bytes.NewBuffer(template))
	req.Header.Add("Authorization", auth)
	if err != nil {
		panic(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var d interface{}
	if err := json.Unmarshal(body, &d); err != nil {
		panic(err)
	}
	iface := d.(map[string]interface{})
	return iface["id"].(string)
}

func pollForStatus(client *http.Client, auth string, requestID string, f *os.File) bool {
	for {
		statusURL := fmt.Sprintf("%s/catalog-service/api/consumer/requests/%s", VraVars.baseURI, requestID)
		req, err := http.NewRequest("GET", statusURL, nil)
		req.Header.Add("Authorization", auth)
		if err != nil {
			panic(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		var d interface{}
		if err := json.Unmarshal(body, &d); err != nil {
			panic(err)
		}
		iface := d.(map[string]interface{})
		str := fmt.Sprintf("Request state is: %s\n", iface["state"].(string))
		f.WriteString(str)
		fmt.Print(str)
		if iface["state"].(string) == "SUCCESSFUL" {
			return true
		}
		if iface["state"].(string) == "FAILED" {
			return false
		}
		time.Sleep(20 * time.Second)
	}
}

func getResource(client *http.Client, auth string, requestID string, f *os.File) {
	url := fmt.Sprintf("%s/catalog-service/api/consumer/resources", VraVars.baseURI)
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", auth)
	if err != nil {
		panic(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var d interface{}
	if err := json.Unmarshal(body, &d); err != nil {
		panic(err)
	}
	iface := d.(map[string]interface{})
	var buffer bytes.Buffer
	for _, itemIf := range iface["content"].([]interface{}) {
		item := itemIf.(map[string]interface{})
		if item["requestId"].(string) == requestID {
			resourceID := item["id"].(string)
			referanceID := item["resourceTypeRef"].(map[string]interface{})["id"].(string)
			label := item["resourceTypeRef"].(map[string]interface{})["label"].(string)
			str := fmt.Sprintf("Resource ID: %s\t Reference ID: %s\t Label: %s\t\n", resourceID, referanceID, label)
			f.WriteString(str)
			fmt.Print(str)
			buffer.WriteString(str)
		}
	}

}

func getClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

func main() {
	setVars()
	f, ferr := os.Create("output_create.log")
	if ferr != nil {
		panic(ferr)
	}
	defer f.Close()
	client := getClient()
	token := getToken(client)
	auth := "Bearer " + token
	catalogURL := VraVars.baseURI + "/catalog-service/api/consumer/entitledCatalogItemViews"
	req, _ := http.NewRequest("GET", catalogURL, nil)
	req.Header.Add("Authorization", auth)
	q := req.URL.Query()
	q.Add("$filter", fmt.Sprintf("name eq '%s'", VraVars.blueprint))
	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var d interface{}
	if err := json.Unmarshal(body, &d); err != nil {
		panic(err)
	}
	iface := d.(map[string]interface{})
	blueprint := iface["content"].([]interface{})[0]
	catalogItem := blueprint.(map[string]interface{})
	catalogID := catalogItem["catalogItemId"].(string)
	str := fmt.Sprintf("Found blueprint: %s with ID: %s\n", VraVars.blueprint, catalogID)
	f.WriteString(str)
	fmt.Print(str)

	// Get the Business Group ID, (needed for template)
	var businessGroupID string
	for _, org := range catalogItem["entitledOrganizations"].([]interface{}) {
		subtenantRef := org.(map[string]interface{})["subtenantRef"].(string)
		subtenantLabel := org.(map[string]interface{})["subtenantLabel"].(string)
		if subtenantLabel == VraVars.businessGroup {
			businessGroupID = subtenantRef
			str = fmt.Sprintf("Found organization: %s, %s\n", subtenantLabel, businessGroupID)
			f.WriteString(str)
			fmt.Print(str)
		}
	}

	// Retrieve template for this blueprint
	template := getTemplate(client, auth, catalogID, businessGroupID)

	// invoke Template
	requestID := invokeTemplate(client, auth, template, catalogID)
	str = fmt.Sprintf("Request ID: %s, Blueprint %s\n", requestID, VraVars.blueprint)
	f.WriteString(str)
	fmt.Print(str)

	success := pollForStatus(client, auth, requestID, f)
	if !success {
		str = fmt.Sprintf("Deployment Failed.  RequestID: %s\n", requestID)
		f.WriteString(str)
		fmt.Print(str)
		os.Exit(1)
	}

	// fetch the resource
	getResource(client, auth, requestID, f)
	os.Exit(0)
}
