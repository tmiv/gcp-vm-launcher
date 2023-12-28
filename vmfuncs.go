package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"text/template"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
)

var (
	funcMap = template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
	}
)

func createInstance(ctx context.Context, data interface{}) error {
	reqTemplate := template.New("vm-template")
	reqTemplateEnv := os.Getenv("VM_REQ_TEMPLATE")
	if len(reqTemplateEnv) < 1 {
		return fmt.Errorf("VM_REQ_TEMPLATE not set")
	}
	reqTemplate, err := reqTemplate.Funcs(funcMap).Parse(reqTemplateEnv)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %w", err)
	}
	var tpl bytes.Buffer
	err = reqTemplate.Option("missingkey=error").Execute(&tpl, data)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %w", err)
	}
	req := computepb.InsertInstanceRequest{}
	json.Unmarshal(tpl.Bytes(), &req)

	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %w", err)
	}
	defer instancesClient.Close()

	op, err := instancesClient.Insert(ctx, &req)
	if err != nil {
		return fmt.Errorf("unable to create instance: %w", err)
	}

	if err = op.Wait(ctx); err != nil {
		return fmt.Errorf("unable to wait for the operation: %w", err)
	}

	log.Printf("Instance created\n")

	return nil
}

func destroyInstance(ctx context.Context, data interface{}) error {
	reqTemplate := template.New("vm-kill-template")
	killTemplateEnv := os.Getenv("VM_KILL_TEMPLATE")
	if len(killTemplateEnv) < 1 {
		return fmt.Errorf("VM_KILL_TEMPLATE not set")
	}
	reqTemplate, err := reqTemplate.Funcs(funcMap).Parse(killTemplateEnv)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %w", err)
	}
	var tpl bytes.Buffer
	err = reqTemplate.Option("missingkey=error").Execute(&tpl, data)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %w", err)
	}

	req := &computepb.DeleteInstanceRequest{}
	json.Unmarshal(tpl.Bytes(), &req)
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %w", err)
	}
	defer instancesClient.Close()
	op, err := instancesClient.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("unable to delete instance: %w", err)
	}

	if err = op.Wait(ctx); err != nil {
		return fmt.Errorf("unable to wait for the operation: %w", err)
	}

	log.Printf("Instance deleted\n")
	return nil
}

func launchVM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("Bad Method %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		log.Printf("Bad Content Type %s\n", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data := make(map[string]interface{})
	postContent, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Bad Post Content %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Body.Close()
	err = json.Unmarshal(postContent, &data)
	if err != nil {
		log.Printf("Bad Post Content Parse %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = createInstance(r.Context(), data)
	if err != nil {
		log.Printf("Launch VM Error %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func killVM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("Bad Method %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		log.Printf("Bad Content Type %s\n", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data := make(map[string]interface{})
	postContent, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Bad Post Content %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Body.Close()
	err = json.Unmarshal(postContent, &data)
	if err != nil {
		log.Printf("Bad Post Content Parse %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = destroyInstance(r.Context(), data)
	if err != nil {
		log.Printf("Kill VM Error %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
