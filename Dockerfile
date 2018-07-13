FROM oraclego 

WORKDIR /go/src/github.com/igknot/gppStandby
ADD database/clientSoftware/oci8_linux.pc /oreclient_install_dir/instantclient_12_2/oci8.pc
RUN rm -fr /go/src/github.com/igknot/gppStandby

WORKDIR /go/src/github.com/igknot

RUN git -c http.sslVerify=false clone -v https://github.com/igknot/gppStandby.git

RUN go get -v ./...

#RUN cp database/clientSoftware/oci8_linux.pc /oreclient_install_dir/instantclient_12_2/oci8.pc

RUN go install -v ./...

#----------------------------
FROM bitnami/minideb

RUN apt-get update
RUN apt-get install libaio1 libaio-dev curl  unzip -y

RUN  mkdir -p /go/bin/
WORKDIR /oreclient_install_dir/

COPY --from=0 /oreclient_install_dir/ /oreclient_install_dir/
RUN ln -s /oreclient_install_dir/instantclient_12_2/libclntsh.so.12.1 /usr/lib/libclntsh.dylib
RUN ln -s /oreclient_install_dir/instantclient_12_2/libclntsh.so.12.1 /usr/lib/libclntsh.so
RUN ln -s /oreclient_install_dir/instantclient_12_2/libocci.so.12.1 /usr/lib/libocci.dylib
RUN ln -s /oreclient_install_dir/instantclient_12_2/libocci.so.12.1 /usr/lib/libocci.so

COPY --from=0 /go/src/github.com/igknot/gppStandby/database/clientSoftware/oci8_linux.pc /oreclient_install_dir/instantclient_12_2/oci8.pc

COPY --from=0 /go/bin/ /go/bin/


ENV PKG_CONFIG_PATH "/oreclient_install_dir/instantclient_12_2"
ENV LD_LIBRARY_PATH "/oreclient_install_dir/instantclient_12_2"

RUN rm -f /oreclient_install_dir/instant*.zip
RUN rm -fr /var/lib/apt/lists

ENV TZ=Africa/Johannesburg
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
ENTRYPOINT /go/bin/gppStandby
