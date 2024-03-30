package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"

    v1 "k8s.io/api/apps/v1"
    admissionv1beta1 "k8s.io/api/admission/v1beta1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/serializer"
)

// Setup global serializer for decoding the AdmissionReview requests
var codecs = serializer.NewCodecFactory(runtime.NewScheme())

func handleAdmissionReview(w http.ResponseWriter, r *http.Request) {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, fmt.Sprintf("could not read request body: %v", err), http.StatusInternalServerError)
        return
    }

    // Parse the AdmissionReview request.
    var admissionReviewReq admissionv1beta1.AdmissionReview
    deserializer := codecs.UniversalDeserializer()
    _, _, err = deserializer.Decode(body, nil, &admissionReviewReq)
    if err != nil {
        http.Error(w, fmt.Sprintf("could not decode request: %v", err), http.StatusBadRequest)
        return
    }

    // Extract the Deployment from the request.
    raw := admissionReviewReq.Request.Object.Raw
    var deployment v1.Deployment
    if err := json.Unmarshal(raw, &deployment); err != nil {
        http.Error(w, fmt.Sprintf("could not unmarshal deployment from request: %v", err), http.StatusBadRequest)
        return
    }

    // Prepare the response object
    response := admissionv1beta1.AdmissionReview{
        Response: &admissionv1beta1.AdmissionResponse{
            UID:     admissionReviewReq.Request.UID,
            Allowed: true,
        },
    }

    // Check if resource requests are specified.
    for _, container := range deployment.Spec.Template.Spec.Containers {
        if container.Resources.Requests == nil {
            response.Response.Allowed = false
            response.Response.Result = &metav1.Status{
                Message: "missing resource requests",
            }
            break
        }
    }

    // Send the response.
    respBytes, err := json.Marshal(response)
    if err != nil {
        http.Error(w, fmt.Sprintf("could not marshal response: %v", err), http.StatusInternalServerError)
        return
    }
    w.Write(respBytes)
}

func main() {
    http.HandleFunc("/validate", handleAdmissionReview)
    fmt.Println("Starting webhook server...")
    if err := http.ListenAndServeTLS(":443", "/etc/webhook/certs/tls.crt", "/etc/webhook/certs/tls.key", nil); err != nil {
        fmt.Fprintf(os.Stderr, "failed to start server: %v\n", err)
        os.Exit(1)
    }
}
