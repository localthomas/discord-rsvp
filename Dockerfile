FROM golang AS builder
WORKDIR /
COPY ./src ./
RUN CGO_ENABLED=0 go build -o /discord-rsvp

FROM scratch
WORKDIR /
COPY --from=builder /discord-rsvp /discord-rsvp
VOLUME /data
# Add root certificate for Discords' API
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs/
EXPOSE 80
ENTRYPOINT [ "/discord-rsvp" ]
