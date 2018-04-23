package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/alibaba/pouch/apis/types"
	"github.com/sirupsen/logrus"
)

func (c ContPlugin) PreCreate(in io.ReadCloser) (io.ReadCloser, error) {
	logrus.Infof("pre create method called")
	inputBuffer, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}
	logrus.Infof("create container with body %s", string(inputBuffer))

	//create network if exist
	var createConfig ContainerCreateConfig
	err = json.NewDecoder(bytes.NewReader(inputBuffer)).Decode(&createConfig)
	if err != nil {
		return nil, err
	}
	if createConfig.HostConfig == nil {
		createConfig.HostConfig = &HostConfig{}
	}
	requestedIP := ""
	if createConfig.HostConfig.NetworkMode == "default" || createConfig.HostConfig.NetworkMode == "" {
		createConfig.HostConfig.NetworkMode = "bridge"
	}
	networkMode := createConfig.HostConfig.NetworkMode
	if err != nil {
		return nil, err
	}
	env := createConfig.Env

	//setup network just in case
	if !strings.HasPrefix(networkMode, "container:") && networkMode != "host" && networkMode != "none" {
		requestedIP = getEnv(env, "RequestedIP")
		defaultRoute := getEnv(env, "DefaultRoute")
		mask := getEnv(env, "DefaultMask")
		nic := getEnv(env, "DefaultNic")
		if createConfig.NetworkingConfig == nil {
			createConfig.NetworkingConfig = &NetworkingConfig{}
		}
		if createConfig.NetworkingConfig.EndpointsConfig == nil {
			createConfig.NetworkingConfig.EndpointsConfig = make(map[string]*EndpointSettings)
		}

		if nwName, e := prepare_network(requestedIP, defaultRoute, mask, nic, networkMode,
			createConfig.NetworkingConfig.EndpointsConfig, env); e != nil {
			return nil, e
		} else if nwName != networkMode {
			createConfig.HostConfig.NetworkMode = nwName
		}

		if mustRequestedIP() {
			if len(requestedIP) == 0 {
				return nil, fmt.Errorf("-e RequestedIP not set")
			}
			for _, oneIp := range strings.Split(requestedIP, ",") {
				if net.ParseIP(oneIp) == nil {
					return nil, fmt.Errorf("-e RequestedIP=%s is invalid", requestedIP)
				}
			}
		}
	}

	if getEnv(createConfig.Env, "ali_admin_uid") == "0" && requestedIP != "" {
		if b, ex := exec.Command("/opt/ali-iaas/pouch/bin/get_admin_uid.sh",
			requestedIP).CombinedOutput(); ex != nil {
			logrus.Errorf("get admin uid error, ip is %s, error is %v", requestedIP, ex)
			return nil, ex
		} else {
			if uid, ex := strconv.Atoi(strings.TrimSpace(string(b))); ex != nil {
				logrus.Errorf("get admin uid error, ip is %s, error is %v", requestedIP, ex)
				return nil, ex
			} else {
				for i, oneEnv := range createConfig.Env {
					arr := strings.SplitN(oneEnv, "=", 2)
					if len(arr) == 2 && arr[0] == "ali_admin_uid" {
						createConfig.Env[i] = fmt.Sprintf("%s=%d", arr[0], uid)
						break
					}
				}
			}
		}
	}

	// common vm must run as root
	mode := getEnv(createConfig.Env, "ali_run_mode")
	if ("common_vm" == mode || "vm" == mode) && createConfig.User != "root" {
		logrus.Infof("in common_vm mode, use root user to start container.")
		createConfig.User = "root"
	}

	// setup disk quota
	if diskQuota := createConfig.Labels["DiskQuota"]; diskQuota != "" &&
		len(createConfig.DiskQuota) == 0 {
		if createConfig.DiskQuota == nil {
			createConfig.DiskQuota = make(map[string]string)
		}
		for _, kv := range strings.Split(diskQuota, ";") {
			kv = strings.TrimSpace(kv)
			if kv == "" {
				continue
			}
			arr := strings.SplitN(kv, "=", 2)
			var k, v string
			if len(arr) == 2 {
				k, v = arr[0], arr[1]
			} else {
				k = ".*"
				v = arr[0]
			}
			createConfig.DiskQuota[k] = v
		}
	}

	// common vm use rich container which introduced by pouch
	if getEnv(env, "ali_run_mode") == "vm" || getEnv(env, "ali_run_mode") == "common_vm" {
		// change common_vm to vm
		for i, line := range createConfig.Env {
			if line == "ali_run_mode=common_vm" {
				createConfig.Env[i] = "ali_run_mode=vm"
			}
		}

		// convert label to env
		for k, v := range createConfig.Labels {
			createConfig.Env = append(createConfig.Env, fmt.Sprintf("%s=%s", escapseLableToEnvName(k), v))
		}

		// generate quota id as needed
		if createConfig.Labels["AutoQuotaId"] == "true" {
			if createConfig.QuotaID == "" || createConfig.QuotaID == "0" {
				qid := createConfig.Labels["QuotaId"]
				if qid != "" && qid != "0" {
					createConfig.QuotaID = qid
				} else {
					createConfig.QuotaID = "-1"
				}
			}
		}

		// set hostname to env
		if getEnv(env, "HOSTNAME") == "" && createConfig.Hostname != "" {
			found := false
			for i, line := range createConfig.Env {
				if strings.HasPrefix(line, "HOSTNAME=") {
					createConfig.Env[i] = fmt.Sprintf("HOSTNAME=%s", createConfig.Hostname)
					found = true
					break
				}
			}
			if !found {
				createConfig.Env = append(createConfig.Env, fmt.Sprintf("HOSTNAME=%s", createConfig.Hostname))
			}
		}
		createConfig.HostConfig.CapAdd = append(createConfig.HostConfig.CapAdd, "SYS_RESOURCE", "SYS_MODULE",
			"SYS_PTRACE", "SYS_PACCT", "NET_ADMIN", "SYS_ADMIN")
	}

	// marshal it as stream and return to the caller
	var out bytes.Buffer
	err = json.NewEncoder(&out).Encode(createConfig)
	logrus.Infof("after process create container body is %s", string(out.Bytes()))

	return ioutil.NopCloser(&out), err
}