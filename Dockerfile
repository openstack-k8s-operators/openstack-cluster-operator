# golang-builder is used in OSBS build
ARG GOLANG_BUILDER=golang:1.16
ARG OPERATOR_BASE_IMAGE=registry.access.redhat.com/ubi7/ubi-minimal:latest

FROM ${GOLANG_BUILDER} AS builder

ARG REMOTE_SOURCE=.
ARG REMOTE_SOURCE_DIR=openstack-cluster-operator
ARG REMOTE_SOURCE_SUBDIR=.
ARG DEST_ROOT=/dest-root
ARG GO_BUILD_EXTRA_ARGS="-v "

COPY $REMOTE_SOURCE $REMOTE_SOURCE_DIR
WORKDIR ${REMOTE_SOURCE_DIR}/${REMOTE_SOURCE_SUBDIR}

RUN mkdir -p ${DEST_ROOT}/usr/local/bin/

# Build
RUN CGO_ENABLED=0 GO111MODULE=on go build ${GO_BUILD_EXTRA_ARGS} -a -o ${DEST_ROOT}/usr/local/bin/manager main.go
RUN CGO_ENABLED=0 GO111MODULE=on go build ${GO_BUILD_EXTRA_ARGS} -a -o ${DEST_ROOT}/usr/local/bin/csv-merger tools/csv-merger/csv-merger.go

RUN cp tools/user_setup ${DEST_ROOT}/usr/local/bin/

# prep the bundle
RUN mkdir -p ${DEST_ROOT}/bundle
RUN cp -r config/crd/bases/* ${DEST_ROOT}/bundle

# strip top 2 lines (this resolves parsing in opm which handles this badly)
RUN sed -i -e 1,2d ${DEST_ROOT}/bundle/*

RUN cp -r bindata $DEST_ROOT/bindata

FROM ${OPERATOR_BASE_IMAGE}
ARG DEST_ROOT=/dest-root

LABEL   com.redhat.component="openstack-cluster-operator-container" \
        name="cn-osp/openstack-cluster-operator" \
        version="0.0.1" \
        summary="OpenStack Cluster Operator" \
        io.k8s.display-name="openstack-cluster-operator" \
        io.k8s.description="This image contains the openstack-cluster" \
        io.openshift.tags="cn-openstack openstack"

ENV USER_UID=1001 \
    OPERATOR_BINDATA_DIR=/bindata/ \
    OPERATOR_BUNDLE=/usr/share/openstack-cluster-operator/bundle/

# install operator binary
COPY --from=builder ${DEST_ROOT}/usr/local/bin/* /usr/local/bin/

# install our bindata
RUN  mkdir -p ${OPERATOR_BINDATA_DIR}
COPY --from=builder $DEST_ROOT/bindata ${OPERATOR_BINDATA_DIR}

# install CRDs and required roles, services, etc
RUN  mkdir -p ${OPERATOR_BUNDLE}
COPY --from=builder ${DEST_ROOT}/bundle/* ${OPERATOR_BUNDLE}

WORKDIR /

# user setup
RUN  /usr/local/bin/user_setup
USER ${USER_UID}

ENTRYPOINT ["/usr/local/bin/manager"]
