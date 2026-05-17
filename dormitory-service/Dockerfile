# ---- Build stage ----
FROM eclipse-temurin:17-jdk AS builder
WORKDIR /build
COPY pom.xml .
COPY src/ src/
RUN apt-get update && apt-get install -y maven && rm -rf /var/lib/apt/lists/* && \
    mvn package -DskipTests -q

# ---- Runtime stage ----
FROM eclipse-temurin:17-jre
WORKDIR /app
COPY --from=builder /build/target/dormitory-service-1.0.0-SNAPSHOT.jar app.jar
EXPOSE 8081
ENTRYPOINT ["java", "-jar", "app.jar"]
