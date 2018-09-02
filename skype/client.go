package skype

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/thehadalone/metachat/metachat"
)

var (
	ppftRegexp     = regexp.MustCompile(`<input.*?name="PPFT".*?value="(.*?)"`)
	locationRegexp = regexp.MustCompile(`(https://[^/]+)/v1/users/ME/endpoints(/%7B[a-z0-9-]+%7D)?`)
	regTokenRegexp = regexp.MustCompile(`(?i)(registrationToken=[a-z0-9+/=]+)`)
	expireRegexp   = regexp.MustCompile(`expires=(\d+)`)
	endpointRegexp = regexp.MustCompile(`endpointId=({[a-z0-9\-]+})`)
)

type (
	httpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	// Config structure.
	Config struct {
		Username    string     `json:"username"`
		Password    string     `json:"password"`
		DisplayName string     `json:"displayName"`
		HTTPClient  httpClient `json:"-"`
	}

	// Client is a Skype client.
	Client struct {
		httpClient                  httpClient
		username                    string
		password                    string
		displayName                 string
		messageChan                 chan metachat.Message
		skypeToken                  string
		registrationToken           string
		registrationTokenExpiration time.Time
		messageHost                 string
		endpointID                  string
	}

	loginParams struct {
		msprequ string
		mspok   string
		ppft    string
	}

	event struct {
		EventMessages []struct {
			Resource resource `json:"resource,omitempty"`
		} `json:"eventMessages,omitempty"`
	}

	resource struct {
		ConversationLink string `json:"conversationLink,omitempty"`
		Imdisplayname    string `json:"imdisplayname,omitempty"`
		Messagetype      string `json:"messagetype"`
		Content          string `json:"content,omitempty"`
	}

	message struct {
		ContentType string `json:"contenttype"`
		MessageType string `json:"messagetype"`
		Content     string `json:"content"`
	}
)

// NewClient is a Skype client constructor.
func NewClient(config Config) (*Client, error) {
	if config.Username == "" || config.Password == "" || config.DisplayName == "" {
		return nil, errors.New("username, password and display name can't be nil")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	client := &Client{
		httpClient:  httpClient,
		username:    config.Username,
		password:    config.Password,
		displayName: config.DisplayName,
		messageChan: make(chan metachat.Message, 100),
		messageHost: "https://client-s.gateway.messenger.live.com",
	}

	return client, nil
}

// Name returns the messenger name.
func (c *Client) Name() string {
	return "Skype"
}

// MessageChan returns a read-only message channel.
func (c *Client) MessageChan() <-chan metachat.Message {
	return c.messageChan
}

// Start starts the client main loop.
func (c *Client) Start() error {
	if time.Now().After(c.registrationTokenExpiration) {
		err := c.getTokens()
		if err != nil {
			return err
		}

		err = c.subscribe()
		if err != nil {
			return err
		}
	}

	for {
		resources, err := c.getMessages()
		if err != nil {
			return err
		}

		for _, r := range resources {
			c.messageChan <- convertToMetachat(r)
		}
	}
}

// Webhook returns HTTP handler for webhook requests.
func (c *Client) Webhook() http.Handler {
	return nil
}

// Send sends a message to chat with the provided ID.
func (c *Client) Send(msg metachat.Message, chat string) error {
	if time.Now().After(c.registrationTokenExpiration) {
		err := c.getTokens()
		if err != nil {
			return err
		}
	}

	payload, err := json.Marshal(convertToSkype(msg))
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/users/ME/conversations/%s/messages",
		c.messageHost, chat), bytes.NewReader(payload))

	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Add("RegistrationToken", c.registrationToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.New("can't send a message, status " + resp.Status)
	}

	return nil
}

func (c *Client) getTokens() error {
	loginParams, err := c.getLoginParams()
	if err != nil {
		return err
	}

	t, err := c.getT(loginParams)
	if err != nil {
		return err
	}

	err = c.getSkypeToken(t)
	if err != nil {
		return err
	}

	err = c.getRegistrationToken()
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getLoginParams() (loginParams, error) {
	req, err := http.NewRequest(http.MethodGet, "https://login.skype.com/login/oauth/microsoft?client_id="+
		"578134&redirect_uri=https://web.skype.com", http.NoBody)

	if err != nil {
		return loginParams{}, errors.WithStack(err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return loginParams{}, errors.WithStack(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return loginParams{}, errors.New("can't get to the Skype login page")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return loginParams{}, errors.WithStack(err)
	}

	ppftMatch := ppftRegexp.FindStringSubmatch(string(body))
	if ppftMatch == nil {
		return loginParams{}, errors.New("can't retrieve PPFT from login form")
	}

	msprequ := getCookieByName(resp.Cookies(), "MSPRequ")
	mspok := getCookieByName(resp.Cookies(), "MSPOK")
	if msprequ == "" || mspok == "" {
		return loginParams{}, errors.New("can't retrieve MSPRequ/MSPOK cookies")
	}

	return loginParams{
		msprequ: msprequ,
		mspok:   mspok,
		ppft:    ppftMatch[1],
	}, nil
}

func (c *Client) getT(params loginParams) (string, error) {
	data := url.Values{}
	data.Set("login", c.username)
	data.Set("passwd", c.password)
	data.Set("PPFT", params.ppft)

	req, err := http.NewRequest(http.MethodPost, "https://login.live.com/ppsecure/post.srf?wa=wsignin1.0&wp="+
		"MBI_SSL&wreply=https://lw.skype.com/login/oauth/proxy?client_id=578134&site_name=lw.skype.com&redirect_uri="+
		"https%3A%2F%2Fweb.skype.com%2F", strings.NewReader(data.Encode()))

	if err != nil {
		return "", errors.WithStack(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.AddCookie(&http.Cookie{Name: "MSPRequ", Value: params.msprequ})
	req.AddCookie(&http.Cookie{Name: "MSPOK", Value: params.mspok})
	req.AddCookie(&http.Cookie{Name: "CkTst", Value: strconv.FormatInt(time.Now().Unix(), 10)})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.WithStack(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("can't get to the Live login page")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", errors.WithStack(err)
	}

	value, ok := doc.Find("#t").Attr("value")
	if !ok {
		return "", errors.New("can't find T field")
	}

	return value, nil
}

func (c *Client) getSkypeToken(t string) error {
	data := url.Values{}
	data.Set("client_id", "578134")
	data.Set("redirect_uri", "https://web.skype.com")
	data.Set("oauthPartner", "999")
	data.Set("site_name", "lw.skype.com")
	data.Set("t", t)

	req, err := http.NewRequest(http.MethodPost, "https://login.skype.com/login/microsoft?client_id="+
		"578134&redirect_uri=https://web.skype.com", strings.NewReader(data.Encode()))

	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("can't get to the Skype login page")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}

	tokenValue, ok := doc.Find("input[name=skypetoken]").Attr("value")
	if !ok {
		return errors.New("can't find token field")
	}

	c.skypeToken = tokenValue

	return nil
}

func (c *Client) getRegistrationToken() error {
	timestamp := time.Now().Unix()
	lockID := "msmsgs@msnmsgr.com"
	hash := skypeHMACSHA256(strconv.FormatInt(timestamp, 10), lockID, "Q1P7W2E4J9R8U3S5")

	req, err := http.NewRequest(http.MethodPost, c.messageHost+"/v1/users/ME/endpoints", bytes.NewReader([]byte("{}")))
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Add("Authentication", "skypetoken="+c.skypeToken)
	req.Header.Add("LockAndKey", fmt.Sprintf("appId=%s; time=%d; lockAndKeyResponse=%s", lockID, timestamp, hash))
	req.Header.Add("BehaviorOverride", "redirectAs404")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusNotFound {

		return errors.Errorf("got %s instead of 200 OK, 201 Created or 404 Not Found", resp.Status)
	}

	location := resp.Header.Get("Location")
	if location != "" {
		groups := locationRegexp.FindStringSubmatch(location)
		if groups == nil {
			return errors.New("unknown Location header format")
		}

		c.endpointID = strings.TrimPrefix(strings.Replace(strings.Replace(groups[2], "%7B", "{", -1), "%7D", "}", -1), "/")
		newMessageHost := groups[1]
		if c.messageHost != newMessageHost {
			c.messageHost = newMessageHost

			return c.getRegistrationToken()
		}
	}

	info := resp.Header.Get("Set-RegistrationToken")
	if info == "" {
		return errors.New("no registration token header")
	}

	regTokenGroups := regTokenRegexp.FindStringSubmatch(info)
	if regTokenGroups == nil {
		return errors.New("no registration token in the header")
	}

	expireGroups := expireRegexp.FindStringSubmatch(info)
	if expireGroups == nil {
		return errors.New("no registration token expire in the header")
	}

	expireValue, err := strconv.ParseInt(expireGroups[1], 10, 64)
	if err != nil {
		return errors.WithStack(err)
	}

	c.registrationToken = regTokenGroups[1]
	c.registrationTokenExpiration = time.Unix(expireValue, 0)

	endpointGroups := endpointRegexp.FindStringSubmatch(info)
	if endpointGroups != nil {
		c.endpointID = endpointGroups[1]
	}

	if c.endpointID == "" && resp.StatusCode == http.StatusOK {
		var data []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			return errors.WithStack(err)
		}

		c.endpointID = data[0]["id"].(string)
	}

	if c.endpointID == "" {
		return errors.New("no endpoint ID in the header")
	}

	return nil
}

func (c *Client) subscribe() error {
	data := map[string]interface{}{
		"template":            "raw",
		"channelType":         "httpLongPoll",
		"interestedResources": []string{"/v1/users/ME/conversations/ALL/messages"},
	}

	reqBody, err := json.Marshal(data)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/users/ME/endpoints/%s/subscriptions",
		c.messageHost, c.endpointID), bytes.NewReader(reqBody))

	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Add("RegistrationToken", c.registrationToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.Errorf("got %s instead of 201 Created", resp.Status)
	}

	return nil
}

func (c *Client) getMessages() ([]resource, error) {
	if time.Now().After(c.registrationTokenExpiration) {
		err := c.getTokens()
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/users/ME/endpoints/%s/subscriptions/0/poll",
		c.messageHost, c.endpointID), http.NoBody)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Add("RegistrationToken", c.registrationToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got %s instead of 200 OK", resp.Status)
	}

	var event event
	err = json.NewDecoder(resp.Body).Decode(&event)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var result []resource
	for _, eventMessage := range event.EventMessages {
		if c.isSupported(eventMessage.Resource) {
			result = append(result, eventMessage.Resource)
		}
	}

	return result, nil
}

func (c *Client) isSupported(resource resource) bool {
	return resource.Content != "" && !strings.Contains(resource.Content, "URIObject") &&
		(resource.Messagetype == "Text" || resource.Messagetype == "RichText") &&
		resource.Imdisplayname != c.displayName
}

func getCookieByName(cookies []*http.Cookie, name string) string {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie.Value
		}
	}

	return ""
}
