$(PKI_DIR):
	mkdir -p $(PKI_DIR)

	# create a certificate authority (CA) that both the client and server trust.
	# The CA is just a public and private key with the public key wrapped up in a self-signed X.509 certificate.
	openssl req \
		-new \
		-x509 \
		-nodes \
		-days 365 \
		-subj '/C=GB/O=Example/OU=TeamA/CN=ca.example.com' \
		-keyout $(PKI_DIR)/ca.key \
		-out $(PKI_DIR)/ca.crt

	# create the server’s key
	openssl genrsa \
		-out $(PKI_DIR)/server.key 2048

	# create a server Certificate Signing Request
	openssl req \
		-new \
		-key $(PKI_DIR)/server.key \
		-subj '/C=GB/O=Example/OU=TeamA/CN=example.com' \
		-out $(PKI_DIR)/server.csr

	# creates the server signed certificate
	openssl x509 \
		-req \
		-in $(PKI_DIR)/server.csr \
		-CA $(PKI_DIR)/ca.crt \
		-CAkey $(PKI_DIR)/ca.key \
		-CAcreateserial \
		-days 365 \
		-out $(PKI_DIR)/server.crt

	# create the client's key
	openssl genrsa \
		-out $(PKI_DIR)/client.key 2048

	# create a client Certificate Signing Request
	openssl req \
		-new \
		-key $(PKI_DIR)/client.key \
		-subj '/CN=user1.example.com' \
		-out $(PKI_DIR)/client.csr

	# creates the client signed certificate
	openssl x509 \
		-req \
		-in $(PKI_DIR)/client.csr \
		-CA $(PKI_DIR)/ca.crt \
		-CAkey $(PKI_DIR)/ca.key \
		-CAcreateserial \
		-days 365 \
		-out $(PKI_DIR)/client.crt

.PHONY: clean-pki
clean-pki:
	rm -fr $(PKI_DIR)

.PHONY: generate-venom-pki
generate-venom-pki:
	rm -fr $(PKI_VAR_FILE)
	# CA PEM
	printf "tlsRootCA: |-\n$$(cat $(PKI_DIR)/ca.crt | sed 's/^/  /')" >> $(PKI_VAR_FILE)
	# Client PEM
	printf "\n" >> $(PKI_VAR_FILE)
	printf "tlsClientCert: |-\n$$(cat $(PKI_DIR)/client.crt | sed 's/^/  /')" >> $(PKI_VAR_FILE)
	# Client Key
	printf "\n" >> $(PKI_VAR_FILE)
	printf "tlsClientKey: |-\n$$(cat $(PKI_DIR)/client.key | sed 's/^/  /')" >> $(PKI_VAR_FILE)
	printf "\n" >> $(PKI_VAR_FILE)
