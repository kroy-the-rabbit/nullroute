package syncer

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/kroy-the-rabbit/nullroute/internal/config"
)

type Engine struct {
	cfg config.Config
}

func NewEngine(cfg config.Config) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) Config() config.Config {
	return e.cfg
}

func (e *Engine) Sync(ctx context.Context, desired []netip.Prefix) error {
	current, err := loadState(e.cfg.StateFile)
	if err != nil {
		return err
	}

	add, del := diff(current, desired)
	log.Printf("sync: desired=%d current=%d add=%d del=%d", len(desired), len(current), len(add), len(del))

	for _, p := range del {
		if err := e.gobgp(ctx, "global", "rib", "-a", familyForPrefix(p), "del", p.String()); err != nil {
			return fmt.Errorf("failed deleting %s: %w", p, err)
		}
	}
	for _, p := range add {
		if err := e.gobgp(ctx, "global", "rib", "-a", familyForPrefix(p), "add", p.String(), "community", e.cfg.BlackholeComm); err != nil {
			return fmt.Errorf("failed adding %s: %w", p, err)
		}
	}

	return saveState(e.cfg.StateFile, desired)
}

func familyForPrefix(prefix netip.Prefix) string {
	if prefix.Addr().Is4() {
		return "ipv4"
	}
	return "ipv6"
}

func (e *Engine) gobgp(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, e.cfg.GoBGPBin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gobgp %v: %w: %s", args, err, string(out))
	}
	return nil
}

func loadState(path string) ([]netip.Prefix, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []netip.Prefix
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line == "" {
			continue
		}
		p, err := netip.ParsePrefix(line)
		if err != nil {
			continue
		}
		out = append(out, p.Masked())
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out, nil
}

func saveState(path string, prefixes []netip.Prefix) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, p := range prefixes {
		if _, err := w.WriteString(p.String() + "\n"); err != nil {
			return err
		}
	}
	return w.Flush()
}

func diff(current, desired []netip.Prefix) (add []netip.Prefix, del []netip.Prefix) {
	currentSet := make(map[netip.Prefix]struct{}, len(current))
	desiredSet := make(map[netip.Prefix]struct{}, len(desired))
	for _, p := range current {
		currentSet[p.Masked()] = struct{}{}
	}
	for _, p := range desired {
		desiredSet[p.Masked()] = struct{}{}
	}

	for p := range desiredSet {
		if _, ok := currentSet[p]; !ok {
			add = append(add, p)
		}
	}
	for p := range currentSet {
		if _, ok := desiredSet[p]; !ok {
			del = append(del, p)
		}
	}
	sort.Slice(add, func(i, j int) bool { return add[i].String() < add[j].String() })
	sort.Slice(del, func(i, j int) bool { return del[i].String() < del[j].String() })
	return add, del
}
