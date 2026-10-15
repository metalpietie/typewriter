package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	qrcode "github.com/skip2/go-qrcode"
)

// PhoneSync lets phones send their notes to Typewriter and get the changes of the computer, over the Wi-Fi.
// It listens only while it is switched on. Phones are added once, with a code from a QR code. Every request carries the key of its phone,
// and the connection is encrypted with a certificate that the computer made for itself. The phone keeps the fingerprint of that certificate.

const (
	syncProtocol    = 1
	defaultSyncPort = 47821
	pairingLife     = 5 * time.Minute
	pairingTries    = 5
	maxBody         = 10 << 20
)

type syncDevice struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	KeyHash string `json:"keyHash"` // the key itself is not kept, only its fingerprint
	Added   string `json:"added"`
	Last    string `json:"last"`
}

// syncState is kept in sync.json in the data folder: not in a project, so a backup of a project does not carry the keys.
type syncState struct {
	Enabled bool         `json:"enabled"`
	CertPEM string       `json:"certPem"`
	KeyPEM  string       `json:"keyPem"`
	Devices []syncDevice `json:"devices"`
}

// DeviceInfo is a phone as the screen shows it.
type DeviceInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Added string `json:"added"`
	Last  string `json:"last"`
}

// SyncStatus is what the Sync section of the screen shows.
type SyncStatus struct {
	On          bool         `json:"on"`
	Port        int          `json:"port"`
	Addresses   []string     `json:"addresses"`
	Devices     []DeviceInfo `json:"devices"`
	Project     ProjectInfo  `json:"project"`
	Computer    string       `json:"computer"`
	Fingerprint string       `json:"fingerprint"`
	Announced   bool         `json:"announced"` // the computer can be found by name on the Wi-Fi
}

// PairingInfo is what the Add a phone window shows.
type PairingInfo struct {
	Code    string `json:"code"`
	Expires int64  `json:"expires"` // unix seconds
	QR      string `json:"qr"`      // a PNG as a data URL
	Payload string `json:"payload"`
}

type pairing struct {
	code    string
	expires time.Time
	tries   int
}

// PhoneSync is the service that the screen calls.
type PhoneSync struct {
	store *Store
	emit  func(name string) // tells the screen that notes changed (set by main)

	mu        sync.Mutex
	state     *syncState
	srv       *http.Server
	ln        net.Listener
	port      int
	pair      *pairing
	mdns      *zeroconf.Server
	announced bool
}

// EventSynced is sent to the screen after a phone changed notes, so the screen reads them again.
const EventSynced = "typewriter:synced"

func NewPhoneSync(store *Store, emit func(string)) *PhoneSync {
	return &PhoneSync{store: store, emit: emit}
}

func (p *PhoneSync) path() string { return filepath.Join(p.store.dir, "sync.json") }

func (p *PhoneSync) load() (*syncState, error) {
	if p.state != nil {
		return p.state, nil
	}
	st := &syncState{}
	if b, err := os.ReadFile(p.path()); err == nil {
		if err := json.Unmarshal(b, st); err != nil {
			return nil, fmt.Errorf("sync.json is broken: %w", err)
		}
	}
	p.state = st
	return st, nil
}

func (p *PhoneSync) save() error {
	b, err := json.MarshalIndent(p.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.path(), b, 0o600)
}

// makeCert makes a certificate for the computer. It is kept, so the phones can keep its fingerprint.
func makeCert() (certPEM, keyPEM string, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "Typewriter", Organization: []string{"MetalPietie"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"typewriter.local", "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})), nil
}

func fingerprintOf(certPEM string) string {
	blk, _ := pem.Decode([]byte(certPEM))
	if blk == nil {
		return ""
	}
	sum := sha256.Sum256(blk.Bytes)
	return hex.EncodeToString(sum[:])
}

// addressRank puts the addresses that are most likely your home network first: 192.168.x.x, then 10.x.x.x, then the rest.
// Link-local addresses (169.254.x.x) are left out: they are what a network card gives itself when it has no network.
func addressRank(ip net.IP) int {
	switch {
	case ip[0] == 192 && ip[1] == 168:
		return 0
	case ip[0] == 10:
		return 1
	default:
		return 2
	}
}

// localAddresses are the addresses of this computer on a home network, the likeliest first.
func localAddresses() []string {
	var ips []net.IP
	ifs, _ := net.Interfaces()
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ip4 := ipn.IP.To4(); ip4 != nil && ip4.IsPrivate() && !ip4.IsLinkLocalUnicast() {
				ips = append(ips, ip4)
			}
		}
	}
	sort.SliceStable(ips, func(i, j int) bool {
		if ri, rj := addressRank(ips[i]), addressRank(ips[j]); ri != rj {
			return ri < rj
		}
		return ips[i].String() < ips[j].String()
	})
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
}

func computerName() string {
	h, err := os.Hostname()
	if err != nil || strings.TrimSpace(h) == "" {
		return "Typewriter"
	}
	return h
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func keyHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Start switches sync on. It listens on the Wi-Fi and announces the computer. It is safe to call it when it is already on.
func (p *PhoneSync) Start() (SyncStatus, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.startLocked(); err != nil {
		return p.statusLocked(), err
	}
	p.state.Enabled = true
	_ = p.save()
	return p.statusLocked(), nil
}

// resume starts sync again when it was on the last time the app ran.
func (p *PhoneSync) resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if st, err := p.load(); err == nil && st.Enabled {
		_ = p.startLocked()
	}
}

func (p *PhoneSync) startLocked() error {
	if p.srv != nil {
		return nil
	}
	st, err := p.load()
	if err != nil {
		return err
	}
	if st.CertPEM == "" {
		c, k, err := makeCert()
		if err != nil {
			return err
		}
		st.CertPEM, st.KeyPEM = c, k
		if err := p.save(); err != nil {
			return err
		}
	}
	cert, err := tls.X509KeyPair([]byte(st.CertPEM), []byte(st.KeyPEM))
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", defaultSyncPort))
	if err != nil { // the usual port is taken: any free one
		if ln, err = net.Listen("tcp", ":0"); err != nil {
			return fmt.Errorf("could not listen: %w", err)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", p.hello)
	mux.HandleFunc("/pair", p.pairHandler)
	mux.HandleFunc("/sync", p.auth(p.syncHandler))
	srv := &http.Server{
		Handler:           mux,
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	p.srv, p.ln = srv, ln
	p.port = ln.Addr().(*net.TCPAddr).Port
	go func() { _ = srv.ServeTLS(ln, "", "") }()
	// announce the computer on the Wi-Fi. If it fails, the QR code and the address still work.
	if m, err := zeroconf.Register(computerName(), "_typewriter._tcp", "local.", p.port,
		[]string{fmt.Sprintf("v=%d", syncProtocol), "fp=" + fingerprintOf(st.CertPEM)[:16]}, nil); err == nil {
		p.mdns, p.announced = m, true
	}
	return nil
}

// Stop switches sync off.
func (p *PhoneSync) Stop() (SyncStatus, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
	if st, err := p.load(); err == nil {
		st.Enabled = false
		_ = p.save()
	}
	return p.statusLocked(), nil
}

func (p *PhoneSync) stopLocked() {
	if p.mdns != nil {
		p.mdns.Shutdown()
		p.mdns, p.announced = nil, false
	}
	if p.srv != nil {
		_ = p.srv.Close()
		p.srv, p.ln, p.port = nil, nil, 0
	}
	p.pair = nil
}

// shutdown closes the server when the app closes. It does not change what the switch remembers.
func (p *PhoneSync) shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

// Status says whether sync is on and which phones are added.
func (p *PhoneSync) Status() SyncStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.statusLocked()
}

func (p *PhoneSync) statusLocked() SyncStatus {
	st, _ := p.load()
	out := SyncStatus{Devices: []DeviceInfo{}, Addresses: []string{}, Project: p.store.projectInfo(), Computer: computerName()}
	if st == nil {
		return out
	}
	out.On = p.srv != nil
	out.Port = p.port
	out.Announced = p.announced
	out.Fingerprint = fingerprintOf(st.CertPEM)
	if out.On {
		out.Addresses = localAddresses()
	}
	for _, d := range st.Devices {
		out.Devices = append(out.Devices, DeviceInfo{ID: d.ID, Name: d.Name, Added: d.Added, Last: d.Last})
	}
	return out
}

// StartPairing makes a code for adding one phone. The QR code holds the address, the fingerprint and the code.
func (p *PhoneSync) StartPairing() (PairingInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.srv == nil {
		return PairingInfo{}, errors.New("switch Sync with phones on first")
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	code := fmt.Sprintf("%06d", n.Int64())
	p.pair = &pairing{code: code, expires: time.Now().Add(pairingLife)}
	payload, _ := json.Marshal(map[string]any{
		"v": syncProtocol, "name": computerName(), "hosts": localAddresses(), "port": p.port,
		"fp": fingerprintOf(p.state.CertPEM), "code": code,
	})
	png, err := qrcode.Encode(string(payload), qrcode.Medium, 320)
	if err != nil {
		return PairingInfo{}, err
	}
	return PairingInfo{Code: code, Expires: p.pair.expires.Unix(), QR: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), Payload: string(payload)}, nil
}

// CancelPairing closes the Add a phone window's code.
func (p *PhoneSync) CancelPairing() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pair = nil
}

// RemoveDevice forgets a phone. It cannot sync again until it is added again.
func (p *PhoneSync) RemoveDevice(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	st, err := p.load()
	if err != nil {
		return err
	}
	kept := st.Devices[:0]
	for _, d := range st.Devices {
		if d.ID != id {
			kept = append(kept, d)
		}
	}
	st.Devices = kept
	return p.save()
}

// ---- the messages ----

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// hello says who the computer is. It tells nothing about the notes.
func (p *PhoneSync) hello(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"name": computerName(), "protocol": syncProtocol, "app": "Typewriter"})
}

func (p *PhoneSync) pairHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "use POST")
		return
	}
	var req struct {
		Code     string `json:"code"`
		DeviceID string `json:"deviceId"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		fail(w, 400, "bad request")
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	pr := p.pair
	if pr == nil || time.Now().After(pr.expires) {
		p.pair = nil
		fail(w, 403, "there is no open code: press Add a phone on the computer")
		return
	}
	if subtle.ConstantTimeCompare([]byte(pr.code), []byte(strings.TrimSpace(req.Code))) != 1 {
		pr.tries++
		if pr.tries >= pairingTries {
			p.pair = nil // too many wrong tries: the code is dead
		}
		fail(w, 403, "the code is wrong")
		return
	}
	id := strings.TrimSpace(req.DeviceID)
	if !sidRe.MatchString(id) {
		fail(w, 400, "bad device id")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "A phone"
	}
	if len([]rune(name)) > 60 {
		name = string([]rune(name)[:60])
	}
	st, err := p.load()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	key := randomToken(32)
	now := time.Now().UTC().Format(time.RFC3339)
	replaced := false
	for i := range st.Devices {
		if st.Devices[i].ID == id { // the same phone is added again: it gets a new key
			st.Devices[i].Name, st.Devices[i].KeyHash, st.Devices[i].Added = name, keyHash(key), now
			replaced = true
		}
	}
	if !replaced {
		st.Devices = append(st.Devices, syncDevice{ID: id, Name: name, KeyHash: keyHash(key), Added: now})
	}
	if err := p.save(); err != nil {
		fail(w, 500, err.Error())
		return
	}
	p.pair = nil // the code works once
	writeJSON(w, 200, map[string]any{"key": key, "computer": computerName(), "project": p.store.projectInfo(), "protocol": syncProtocol})
}

// auth checks "Authorization: Bearer <device id>:<key>".
func (p *PhoneSync) auth(next func(http.ResponseWriter, *http.Request, syncDevice)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		id, key, ok := strings.Cut(h, ":")
		if !ok || id == "" || key == "" {
			fail(w, 401, "not added")
			return
		}
		p.mu.Lock()
		st, err := p.load()
		var dev *syncDevice
		if err == nil {
			for i := range st.Devices {
				if st.Devices[i].ID == id && subtle.ConstantTimeCompare([]byte(st.Devices[i].KeyHash), []byte(keyHash(key))) == 1 {
					d := st.Devices[i]
					dev = &d
				}
			}
		}
		p.mu.Unlock()
		if dev == nil {
			fail(w, 401, "this phone is not added to this computer")
			return
		}
		next(w, r, *dev)
	}
}

func (p *PhoneSync) syncHandler(w http.ResponseWriter, r *http.Request, dev syncDevice) {
	if r.Method != http.MethodPost {
		fail(w, 405, "use POST")
		return
	}
	var req PhoneRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&req); err != nil {
		fail(w, 400, "bad request")
		return
	}
	req.Device, req.DeviceName = dev.ID, dev.Name // the name that you saw when you added the phone
	resp, err := p.store.syncNotes(req)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	p.mu.Lock()
	if st, err := p.load(); err == nil {
		for i := range st.Devices {
			if st.Devices[i].ID == dev.ID {
				st.Devices[i].Last = time.Now().UTC().Format(time.RFC3339)
			}
		}
		_ = p.save()
	}
	p.mu.Unlock()
	if len(req.Notes) > 0 && p.emit != nil {
		p.emit(EventSynced) // the screen reads the notes again
	}
	writeJSON(w, 200, resp)
}
