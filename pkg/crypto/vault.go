package crypto

import (
	"context"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	vaultapi "github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

// VaultConfig describes the configuration for a Vault instance.
type VaultConfig struct {
	Addr       string `required:"true"`
	TokenMount string `required:"true" split_words:"true"`
	CaCrt      string `split_words:"true"`
	ServerName string `split_words:"true"`
}

// EnsureTokenUpdated enters an infinite loop that watches the given target file and updates the
// token associated with the client whenever the token changes.
func EnsureTokenUpdated(
	ctx context.Context, client *vaultapi.Client, target string, logger *zap.Logger,
) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("failed to initialize file watcher", zap.Error(err))
		return
	}
	defer watcher.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case event, ok := <-watcher.Events:
				if !ok {
					break loop
				}
				if (event.Op&fsnotify.Write > 0) || (event.Op&fsnotify.Create > 0) {
					setTokenFromFile(client, target, logger)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					break loop
				}
				logger.Warn("received error from token watcher", zap.Error(err))
			}
		}
		wg.Done()
	}()

	for i := 0; i < 16; i++ {
		if err := watcher.Add(target); err != nil {
			logger.Error("failed to add file to file watcher, retrying", zap.Error(err))
			select {
			case <-ctx.Done():
				break
			case <-time.After(time.Second << i):
				continue
			}
		}
		setTokenFromFile(client, target, logger)
		break
	}
	wg.Wait()
}

func setTokenFromFile(client *vaultapi.Client, target string, logger *zap.Logger) {
	content, err := ioutil.ReadFile(target)
	if err != nil {
		logger.Warn("failed to read file containing client token", zap.Error(err))
	} else {
		client.SetToken(strings.Trim(string(content), " \n\t"))
		logger.Info("successfully set new client token")
	}
}
