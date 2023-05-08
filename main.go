package main

import (
	"context"
  	"fmt"
  	"net/http"
  	"os"
	"os/signal"
	"syscall"
	"time"

  	"github.com/Ferlab-Ste-Justine/etcd-sdk/client"
  	"github.com/gin-gonic/gin"
)

type Shutdown func() error

func Serve(config Config, doneCh <-chan struct{}) <-chan error {
	var cli *client.EtcdClient
	var server *http.Server
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)

	shutdown := func() error {
		defer cancel()
		
		if server != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)	
			defer shutdownCancel()
			shutdownErr := server.Shutdown(shutdownCtx)
			if shutdownErr != nil && shutdownErr != http.ErrServerClosed {
				return shutdownErr
			}
		}

		return nil
	}

	go func() {
		defer func() {
			errCh <- shutdown()
			close(errCh)
		}()

		var err error
		cli, err = client.Connect(ctx, client.EtcdClientOptions{
			ClientCertPath:    config.EtcdClient.Auth.ClientCert,
			ClientKeyPath:     config.EtcdClient.Auth.ClientKey,
			CaCertPath:        config.EtcdClient.Auth.CaCert,
			Username:          config.EtcdClient.Auth.Username,
			Password:          config.EtcdClient.Auth.Password,
			EtcdEndpoints:     config.EtcdClient.Endpoints,
			ConnectionTimeout: config.EtcdClient.ConnectionTimeout,
			RequestTimeout:    config.EtcdClient.RequestTimeout,
			RetryInterval:     config.EtcdClient.RetryInterval,
			Retries:           config.EtcdClient.Retries,
		})
	
		if err != nil {
			errCh <- err
			return
		}

		if !config.Server.DebugMode {
			gin.SetMode(gin.ReleaseMode)
		}

		accounts, accountsErr := getAccounts(config)
		if accountsErr != nil {
			errCh <- accountsErr
			return	
		}

		router := gin.Default()
		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", config.Server.Address, config.Server.Port),
			Handler: router,
		}
	
		handlers, terminateCh := GetHandlers(config, cli)

		if len(accounts) > 0 {
			authorized := router.Group("/", gin.BasicAuth(accounts))
			authorized.PUT("/lock", handlers.AcquireLock)
			authorized.DELETE("/lock", handlers.ReleaseLock)
			authorized.GET("/state", handlers.GetState)
			authorized.PUT("/state", handlers.UpsertState)
			authorized.DELETE("/state", handlers.DeleteState)
			if config.RemoteTerminiation {
				authorized.POST("/termination", handlers.Terminate)
			}
		} else {
			router.PUT("/lock", handlers.AcquireLock)
			router.DELETE("/lock", handlers.ReleaseLock)
			router.GET("/state", handlers.GetState)
			router.PUT("/state", handlers.UpsertState)
			router.DELETE("/state", handlers.DeleteState)
			if config.RemoteTerminiation {
				router.POST("/termination", handlers.Terminate)
			}
		}

		serverDoneCh := make(chan error)
		go func() {
			defer close(serverDoneCh)
			if config.Server.Tls.Certificate == "" {
				serverErr := server.ListenAndServe()
				if serverErr != nil && serverErr != http.ErrServerClosed {
					serverDoneCh <- serverErr
				}
			} else {
				serverErr := server.ListenAndServeTLS(config.Server.Tls.Certificate, config.Server.Tls.Key)
				if serverErr != nil && serverErr != http.ErrServerClosed {
					serverDoneCh <- serverErr
				}
			}
		}()

		select{
		case <-terminateCh:
		case serverErr := <-serverDoneCh:
			if serverErr != nil {
				errCh <- serverErr
			}
		case <-doneCh:
		}
	}()

	return errCh
}

func main() {
	config, configErr := getConfig()
	if configErr != nil {
		fmt.Println(configErr.Error())
		os.Exit(1)	
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	errCh := Serve(config, doneCh)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigChan
		fmt.Printf("Caught signal %s. Terminating.\n", sig.String())
		doneCh <- struct{}{}
		err := <-errCh
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		os.Exit(0)
	}()

	err := <-errCh
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)	
	}
}