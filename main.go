package main

import (
	"context"
  	"fmt"
  	"net/http"
  	"os"
	"time"

  	"github.com/Ferlab-Ste-Justine/etcd-sdk/client"
  	"github.com/gin-gonic/gin"
)

func main() {
	config, configErr := getConfig()
	if configErr != nil {
		fmt.Println(configErr.Error())
		os.Exit(1)	
	}

	cli, cliErr := client.Connect(client.EtcdClientOptions{
		ClientCertPath:    config.EtcdClient.Auth.ClientCert,
		ClientKeyPath:     config.EtcdClient.Auth.ClientKey,
		CaCertPath:        config.EtcdClient.Auth.CaCert,
		EtcdEndpoints:     config.EtcdClient.Endpoints,
		ConnectionTimeout: config.EtcdClient.ConnectionTimeout,
		RequestTimeout:    config.EtcdClient.RequestTimeout,
		Retries:           config.EtcdClient.Retries,
	})

	if cliErr != nil {
		fmt.Println(cliErr.Error())
		os.Exit(1)	
	}
	defer cli.Close()

	if !config.Server.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	accounts, accountsErr := getAccounts(config)
	if accountsErr != nil {
		fmt.Println(accountsErr.Error())
		os.Exit(1)	
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

	go func() {
		if config.Server.Tls.Certificate == "" {
			serverErr := server.ListenAndServe()
			if serverErr != nil && serverErr != http.ErrServerClosed {
				fmt.Println(serverErr.Error())
				os.Exit(1)	
			}
		} else {
			serverErr := server.ListenAndServeTLS(config.Server.Tls.Certificate, config.Server.Tls.Key)
			if serverErr != nil && serverErr != http.ErrServerClosed {
				fmt.Println(serverErr.Error())
				os.Exit(1)	
			}
		}

	}()

	<-terminateCh
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownErr := server.Shutdown(ctx)
	if shutdownErr != nil {
		fmt.Println(shutdownErr.Error())
		os.Exit(1)
	}

	os.Exit(0)

}