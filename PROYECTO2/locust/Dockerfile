FROM locustio/locust:2.20.0
COPY locustfile.py /home/locust/locustfile.py
WORKDIR /home/locust
EXPOSE 8089
ENTRYPOINT ["locust", "-f", "locustfile.py"]