package sources

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kroy-the-rabbit/nullroute/internal/config"
)

func FetchAndParse(ctx context.Context, cfg config.Config) ([]netip.Prefix, error) {
	client := &http.Client{Timeout: 45 * time.Second}
	type result struct {
		prefixes []netip.Prefix
		err      error
	}

	results := make(chan result, len(cfg.Sources))
	wg := sync.WaitGroup{}

	for _, src := range cfg.Sources {
		s := src
		wg.Add(1)
		go func() {
			defer wg.Done()
			pfx, err := fetchOne(ctx, client, s)
			results <- result{prefixes: pfx, err: err}
		}()
	}

	wg.Wait()
	close(results)

	uniq := make(map[netip.Prefix]struct{})
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		for _, p := range res.prefixes {
			if shouldInclude(p, cfg) {
				uniq[p.Masked()] = struct{}{}
			}
		}
	}

	for _, a := range cfg.AllowlistCIDRs {
		prefix, err := parseToPrefix(a)
		if err != nil {
			return nil, fmt.Errorf("invalid allowlist cidr %q: %w", a, err)
		}
		delete(uniq, prefix.Masked())
	}

	out := make([]netip.Prefix, 0, len(uniq))
	for p := range uniq {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})

	return out, nil
}

func fetchOne(ctx context.Context, client *http.Client, src config.Source) ([]netip.Prefix, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("source %s failed: %w", src.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("source %s returned status %d", src.Name, resp.StatusCode)
	}
	return parsePrefixes(resp.Body)
}

func parsePrefixes(r io.Reader) ([]netip.Prefix, error) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []netip.Prefix
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		line = stripInlineComment(line)
		if line == "" {
			continue
		}
		prefix, err := parseToPrefix(line)
		if err != nil {
			continue
		}
		out = append(out, prefix)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseToPrefix(token string) (netip.Prefix, error) {
	if p, err := netip.ParsePrefix(token); err == nil {
		return p.Masked(), nil
	}

	ip := net.ParseIP(token)
	if ip == nil {
		return netip.Prefix{}, fmt.Errorf("invalid prefix or ip: %q", token)
	}

	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return netip.Prefix{}, fmt.Errorf("invalid ip: %q", token)
	}
	if addr.Is4() {
		return netip.PrefixFrom(addr, 32), nil
	}
	return netip.PrefixFrom(addr, 128), nil
}

func stripInlineComment(s string) string {
	if idx := strings.IndexByte(s, '#'); idx >= 0 {
		s = s[:idx]
	}
	if idx := strings.Index(s, " //"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

func shouldInclude(prefix netip.Prefix, cfg config.Config) bool {
	if prefix.Addr().Is4() {
		return prefix.Bits() >= cfg.MinPrefixV4
	}
	return prefix.Bits() >= cfg.MinPrefixV6
}
