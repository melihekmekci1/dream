package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"

    v1 "k8s.io/api/apps/v1"
    admissionv1 "k8s.io/api/admission/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/serializer"
)

var codecs = serializer.NewCodecFactory(runtime.NewScheme())

func handleAdmissionReview(w http.ResponseWriter, r *http.Request) {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, fmt.Sprintf("could not read request body: %v", err), http.StatusInternalServerError)
        return
    }

    var admissionReviewReq admissionv1.AdmissionReview
    deserializer := codecs.UniversalDeserializer()
    _, _, err = deserializer.Decode(body, nil, &admissionReviewReq)
    if err != nil {
        http.Error(w, fmt.Sprintf("could not decode request: %v", err), http.StatusBadRequest)
        return
    }

    raw := admissionReviewReq.Request.Object.Raw
    var deployment v1.Deployment
    if err := json.Unmarshal(raw, &deployment); err != nil {
        http.Error(w, fmt.Sprintf("could not unmarshal deployment from request: %v", err), http.StatusBadRequest)
        return
    }

    response := admissionv1.AdmissionReview{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "admission.k8s.io/v1",
            Kind:       "AdmissionReview",
        },
        Response: &admissionv1.AdmissionResponse{
            UID:     admissionReviewReq.Request.UID,
            Allowed: true,
        },
    }

    for _, container := range deployment.Spec.Template.Spec.Containers {
        if container.Resources.Requests == nil {
            response.Response.Allowed = false
            response.Response.Result = &metav1.Status{
                Message: "missing resource requests",
            }
            break
        }
    }

    respBytes, err := json.Marshal(response)
    if err != nil {
        http.Error(w, fmt.Sprintf("could not marshal response: %v", err), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
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
