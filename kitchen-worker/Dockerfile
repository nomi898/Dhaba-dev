
# ==========================================
# Stage 1: Build the Spring Boot application
# ==========================================
FROM maven:3.9-eclipse-temurin-21 AS build

WORKDIR /build

# Copy Maven configuration first for dependency caching
COPY pom.xml checkstyle.xml ./

# Download dependencies
RUN mvn -B dependency:go-offline

# Copy application source code
COPY src ./src

# Build executable Spring Boot JAR
RUN mvn -B package

# ==========================================
# Stage 2: Lightweight runtime image
# ==========================================
FROM eclipse-temurin:21-jre-alpine AS runtime

# Create a non-root user
RUN addgroup -S app && adduser -S -G app -u 10001 app

WORKDIR /app

# Copy only the executable JAR from build stage
COPY --from=build --chown=app:app \
    /build/target/kitchen-worker.jar ./app.jar

# Worker heartbeat file
ENV WORKER_HEARTBEAT_FILE=/tmp/kitchen-worker-heartbeat

# Run as non-root user
USER app

HEALTHCHECK --interval=15s --timeout=3s --start-period=40s --retries=3 \
    CMD [ "sh", "-c", "[ -f \"$WORKER_HEARTBEAT_FILE\" ] && [ $(( $(date +%s) - $(stat -c %Y \"$WORKER_HEARTBEAT_FILE\") )) -lt 30 ]" ]
# Start the worker
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "-XX:+ExitOnOutOfMemoryError", "-jar", "/app/app.jar"]
