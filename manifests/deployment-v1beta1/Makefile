SHELL = bash

OUTPUT := pizza-crd.yaml serving-cert-secret.yaml mutatingadmissionregistration.yaml validatingadmissionregistration.yaml

all: tls.key tls.crt $(OUTPUT)

tls.key tls.crt:
		openssl req -new -x509 -subj "/CN=webhook.pizza-crd.svc" -nodes -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365

%.yaml: tls.key tls.crt %.yaml.template
		sed 's,CERT,$(shell base64 tls.crt),;s,KEY,$(shell base64 tls.key),' $@.template > $@

clean:
		rm -f tls.key tls.crt $(OUTPUT)
