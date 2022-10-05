package main

import (
  "fmt"
  "net/http"
  "os"
  "strconv"
  "strings"

  "github.com/Ferlab-Ste-Justine/etcd-sdk/client"
  "github.com/Ferlab-Ste-Justine/etcd-sdk/keymodels"
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
		EtcdEndpoints:     strings.Split(config.EtcdClient.Endpoints, ","),
		ConnectionTimeout: config.EtcdClient.ConnectionTimeout,
		RequestTimeout:    config.EtcdClient.RequestTimeout,
		Retries:           config.EtcdClient.Retries,
	})

	if cliErr != nil {
		fmt.Println(cliErr.Error())
		os.Exit(1)	
	}
	defer cli.Close()

	router := gin.Default()

	router.PUT("/lock", func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}
		
		leaseTtlStr := c.DefaultQuery("lease_ttl", "600")
		ttl, ttlErr := strconv.ParseInt(leaseTtlStr, 10, 64)
		if ttlErr != nil {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "Lease ttl needs to be in integer format",
			})
			return		
		}
		
		state = fmt.Sprintf("%s/lock", state)
		_, alreadyLocked, lockErr := cli.AcquireLock(client.AcquireLockOptions{
			Key: state,
			Ttl: ttl,
			Timeout: config.Lock.Timeout,
			RetryInterval: config.Lock.RetryInterval,
		})
		if alreadyLocked {
			c.JSON(http.StatusLocked, gin.H{
				"status": "locked",
			})
			return		
		}
		if lockErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": lockErr.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	router.DELETE("/lock", func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}

		state = fmt.Sprintf("%s/lock", state)
		releaseErr := cli.ReleaseLock(state)
		if releaseErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": releaseErr.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	//TODO
	router.GET("/state", func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}

		state = fmt.Sprintf("%s/state", state)
		c.JSON(http.StatusOK, gin.H{
			"state": state,
		})
	})

	router.PUT("/state", func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}

		state = fmt.Sprintf("%s/state", state)
		putErr := cli.PutChunkedKey(&keymodels.ChunkedKeyPayload{
			Key: state,
			Value: c.Request.Body,
			Size: c.Request.ContentLength,
		})
		if putErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": putErr.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"state": state,
		})
	})

	router.DELETE("/state", func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}

		state = fmt.Sprintf("%s/state", state)
		deleteErr := cli.DeleteChunkedKey(state)
		if deleteErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": deleteErr.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"state": state,
		})
	})

  	router.Run(fmt.Sprintf("%s:%d", config.Server.Address, config.Server.Port))
}