package webhook

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// ValidateURL confere a URL de webhook antes de gravar: so http/https e com host. Redes
// privadas sao permitidas de proposito (o webhook costuma ir para outro container ou para a
// rede interna); o que nunca e permitido fica no dialer (blockedIP), na hora de conectar.
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("webhook URL must start with http:// or https://")
	}
	if u.Hostname() == "" {
		return errors.New("webhook URL needs a host")
	}
	if u.User != nil {
		return errors.New("webhook URL must not carry credentials; use GATEWAY_META_APP_SECRET (HMAC) to authenticate")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && blockedIP(ip) {
		return fmt.Errorf("webhook URL points to a blocked address (%s)", ip)
	}
	return nil
}

// blockedIP: enderecos que um webhook nunca deve alcancar. Link-local inclui o servico de
// metadados das nuvens (169.254.169.254, fd00:ec2::254), que entrega credenciais da
// maquina: e o alvo classico de SSRF.
func blockedIP(ip net.IP) bool {
	return ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() ||
		ip.IsUnspecified() || ip.Equal(net.ParseIP("fd00:ec2::254"))
}

// safeClient e o cliente HTTP dos webhooks: confere o IP de verdade na hora de conectar,
// depois da resolucao de DNS (um nome que resolve para 169.254.169.254 tambem e barrado,
// inclusive se o DNS mudar depois do cadastro). Nao segue redirect: um 302 para a rede
// interna seria outro jeito de furar a regra.
func safeClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if ip := net.ParseIP(host); ip != nil && blockedIP(ip) {
				return fmt.Errorf("webhook to blocked address %s refused", ip)
			}
			return nil
		},
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           func(ctx context.Context, n, a string) (net.Conn, error) { return dialer.DialContext(ctx, n, a) },
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		MaxIdleConns:          50,
		IdleConnTimeout:       90 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
