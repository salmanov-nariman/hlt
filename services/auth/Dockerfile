FROM maven:3.9-eclipse-temurin-21 AS build
WORKDIR /app/services/auth

COPY services/auth/pom.xml ./pom.xml
RUN mvn dependency:go-offline -B

COPY api/email /app/api/email
COPY services/auth/src ./src
RUN mvn package -DskipTests -B

FROM eclipse-temurin:21-jre
WORKDIR /app

LABEL org.opencontainers.image.source="https://github.com/salmanov-nariman/hlt"

COPY --from=build /app/services/auth/target/*.jar app.jar

EXPOSE 8080
ENTRYPOINT ["java", "-jar", "app.jar"]