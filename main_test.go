package main

import (
	"os"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/Ferlab-Ste-Justine/etcd-sdk/testutils"
)

func TestServe(t *testing.T) {
	currDir, currDirErr := os.Getwd()
	if currDirErr != nil {
		t.Errorf("Error obtaining current working directory: %s", currDirErr.Error())
		return
	}

	testDir := path.Join(currDir, "test")
	relCertsDir := path.Join("certificates", "certs")
	absCertsDir := path.Join(testDir, relCertsDir)
	terrWorkDir := path.Join(testDir, "client")
	outputDir := path.Join(terrWorkDir, "output")

	tearDown, launchErr := testutils.LaunchTestEtcdCluster(testDir, testutils.EtcdTestClusterOpts{
		CaCertPath: path.Join(absCertsDir, "etcd-ca.crt"),
		ServerCertPath: path.Join(absCertsDir, "etcd-server.crt"),
		ServerKeyPath: path.Join(absCertsDir, "etcd-server.key"),
	})
	if launchErr != nil {
		t.Errorf("Error occured launching test etcd cluster: %s", launchErr.Error())
		return
	}

	defer func() {
		errs := tearDown()
		if len(errs) > 0 {
			t.Errorf("Errors occured tearing down etcd cluster: %s", errs[0].Error())
		}
	}()

	doneCh := make(chan struct{})
	errCh := Serve(Config{
		EtcdClient: ConfigEtcdClient{
			Endpoints: []string{
				"127.0.0.1:3379",
				"127.0.0.2:3379",
				"127.0.0.3:3379",
			},
			ConnectionTimeout: 10 * time.Second,
			RequestTimeout: 10 * time.Second,
			RetryInterval: 1 * time.Second,
			Retries: 6,
			Auth: ConfigEtcdAuth{
				CaCert: path.Join(absCertsDir, "etcd-ca.crt"),
				ClientCert: path.Join(absCertsDir, "etcd-root.crt"),
				ClientKey: path.Join(absCertsDir, "etcd-root.key"),
			},
		},
		Lock: ConfigLock{
			Timeout: 5 * time.Second,
			RetryInterval: 1 * time.Second,
		},
		Server: ConfigServer{
			Port: 8080,
			Address: "127.0.0.1",
			DebugMode: false,
		},
		LegacySupport: ConfigLegacySupport{
			Read: false,
			Clear: false,
			AddSlash: false,
		},
		RemoteTerminiation: false,
	}, doneCh)

	defer func() {
		close(doneCh)
		err := <- errCh
		if err != nil {
			t.Errorf("Errors occured closing http server: %s", err.Error())
		}
	}()

	select {
	case err := <-errCh:
		t.Errorf("Errors occured launching http server: %s", err.Error())
		return
	default:
	}

	waitcli := http.Client{}
	res, getErr := waitcli.Get("http://127.0.0.1:8080/health")
	if getErr == nil {
		res.Body.Close()
	}

	waitIdx := 0
	for getErr != nil || res.StatusCode != 200 {
		res, getErr = waitcli.Get("http://127.0.0.1:8080/health")
		if getErr == nil {

			res.Body.Close()
		}
		
		time.Sleep(500*time.Millisecond)
		waitIdx += 1
		if waitIdx == 100 {
			t.Errorf("Waiting too long on http server. Aborting")
			return
		}
	}

	largeString := strings.Repeat("aaa", 10)//1024*1024)
    writeErr := os.WriteFile(path.Join(terrWorkDir, "large_file.txt"), []byte(largeString), 0700)
	if writeErr != nil {
		t.Errorf("Error occured creating a large file: %s", writeErr.Error())
		return
	}

	cmdErr := TerraformApply(terrWorkDir, t)
	if cmdErr != nil {
		t.Errorf("Error occured running terraform apply command: %s", cmdErr.Error())
		return
	}

	defer func() {
		removalErr := os.RemoveAll(outputDir)
		if removalErr != nil {
			t.Errorf("Error occured cleaning up output directory: %s", removalErr.Error())
		}

		removalErr = os.RemoveAll(path.Join(terrWorkDir, ".terraform"))
		if removalErr != nil {
			t.Errorf("Error occured cleaning up terraform dependencies directory: %s", removalErr.Error())
		}

		removalErr = os.RemoveAll(path.Join(terrWorkDir, ".terraform.lock.hcl"))
		if removalErr != nil {
			t.Errorf("Error occured cleaning up terraform lock file: %s", removalErr.Error())
		}
	}()

	exists, existsErr := PathExists(path.Join(terrWorkDir, "terraform.tfstate"))
	if existsErr != nil {
		t.Errorf("Error occured while trying to determine if a file exists: %s", existsErr.Error())
		return
	}
	if exists {
		t.Errorf("Detected a local terraform state. It should have been in etcd")
		return
	}

	exists, existsErr = PathExists(path.Join(terrWorkDir, "terraform.tfstate.backup"))
	if existsErr != nil {
		t.Errorf("Error occured while trying to determin if a file exists: %s", existsErr.Error())
		return
	}
	if exists {
		t.Errorf("Detected a local terraform state. It should have been in etcd")
		return
	}

	exists, existsErr = PathExists(path.Join(outputDir, "large_file_copy.txt"))
	if existsErr != nil {
		t.Errorf("Error occured while trying to determine if a file exists: %s", existsErr.Error())
		return
	}
	if !exists {
		t.Errorf("A large file that should have existed if terraform apply ran successfully didn't exist")
		return
	}

	exists, existsErr = PathExists(path.Join(outputDir, "password"))
	if existsErr != nil {
		t.Errorf("Error occured while trying to determine if a file exists: %s", existsErr.Error())
		return
	}
	if !exists {
		t.Errorf("A password file that should have existed if terraform apply ran successfully didn't exist")
		return
	}

	password, passwordErr := os.ReadFile(path.Join(outputDir, "password"))
	if passwordErr != nil {
		t.Errorf("Error occured retrieving content of the password file: %s", passwordErr.Error())
		return
	}

	cmdErr = TerraformApply(terrWorkDir, t)
	if cmdErr != nil {
		t.Errorf("Error occured running terraform apply command: %s", cmdErr.Error())
		return
	}

	passwordv2, passwordv2Err := os.ReadFile("./test/client/output/password")
	if passwordv2Err != nil {
		t.Errorf("Error occured retrieving content of the password file: %s", passwordv2Err.Error())
		return
	}

	if string(passwordv2) != string(password) {
		t.Errorf("Password value has changed between terraform applies which it wouldn't have if the state has properly been persisted")
		return
	}

	cmdErr = TerraformDestroy(terrWorkDir, t)
	if cmdErr != nil {
		t.Errorf("Error occured running terraform destroy command: %s", cmdErr.Error())
		return
	}

	exists, existsErr = PathExists("./test/client/output/large_file_copy.txt")
	if existsErr != nil {
		t.Errorf("Error occured while trying to determine if a file exists: %s", existsErr.Error())
		return
	}
	if exists {
		t.Errorf("A large file that should have not existed if terraform destroy ran successfully did exist")
		return
	}

	exists, existsErr = PathExists("./test/client/output/password")
	if existsErr != nil {
		t.Errorf("Error occured while trying to determine if a file exists: %s", existsErr.Error())
		return
	}
	if exists {
		t.Errorf("A password file that should not have existed if terraform destroy ran successfully did exist")
		return
	}
}