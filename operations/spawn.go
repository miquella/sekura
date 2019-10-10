package operations

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/aws/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/miquella/sekura/credentials"
)

// Spawn contains all options and logic for spawning commands
type Spawn struct {
	Command   []string
	VaultName string

	Assume []string
	// Refresh bool
	// Region string
}

// Run executes the spawn operation, invoking the given command
func (s *Spawn) Run() error {
	// start the metadata service
	vars := s.getCurrentEnv()
	s.startMetadataService(vars)

	// start the command
	cmd := exec.Command(s.Command[0], s.Command[1:]...)
	cmd.Env = s.buildEnviron(vars)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *Spawn) startMetadataService(vars map[string]string) error {
	config, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}

	// setup all of the assume role providers
	credsProvider := config.Credentials
	for _, role := range s.Assume {
		assumeConfig := config.Copy()
		assumeConfig.Credentials = credsProvider

		assumeClient := sts.New(assumeConfig)
		assumeProvider := stscreds.NewAssumeRoleProvider(assumeClient, role)
		assumeProvider.ExpiryWindow = 30 * time.Second

		credsProvider = assumeProvider
	}

	// setup the listener and http handler
	listener, err := net.Listen("tcp", "localhost:")
	if err != nil {
		return err
	}

	handler, err := credentials.NewHandler(credsProvider)
	if err != nil {
		return err
	}

	// TODO: FIGURE OUT HOW TO CLEANUP THE LISTENER
	go http.Serve(listener, handler)

	// inject necessary variables to the vars map
	vars["AWS_CONTAINER_CREDENTIALS_FULL_URI"] = fmt.Sprintf("http://%s/", listener.Addr())
	vars["AWS_CONTAINER_AUTHORIZATION_TOKEN"] = handler.GetAuthToken()

	delete(vars, "AWS_ACCESS_KEY_ID")
	delete(vars, "AWS_SECRET_ACCESS_KEY")
	delete(vars, "AWS_SECURITY_TOKEN")
	delete(vars, "AWS_SESSION_TOKEN")

	return nil
}

func (s *Spawn) getCurrentEnv() map[string]string {
	environ := os.Environ()
	envs := make(map[string]string)
	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		envs[parts[0]] = parts[1]
	}
	return envs
}

func (s *Spawn) buildEnviron(vars map[string]string) []string {
	environ := []string{}
	for k, v := range vars {
		environ = append(environ, fmt.Sprintf("%s=%s", k, v))
	}
	return environ
}
