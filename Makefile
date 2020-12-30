codegen: controller-gen
	$(CONTROLLER_GEN) \
		object \
		crd:trivialVersions=true \
		rbac:roleName="\"{{ .Release.Name }}\"" \
		paths="./..." \
		output:crd:artifacts:config=deploy/crds \
		output:rbac:artifacts:config=deploy/templates
	for file in deploy/crds/*.yaml; do \
		mv $$file $$(echo $$file | sed -e 's/\(.*\)\/meerkat\.borchero\.com_\(.*\)$$/\1\/\2/g'); \
	done

#--------------------------------------------------------------------------------------------------

controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
