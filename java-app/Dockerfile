# Stage 1: Build the application
# Use a specific version of Maven and JDK to ensure consistency
FROM maven:3.8.4-openjdk-17 as build

# Set the working directory in the builder stage
WORKDIR /app

# Copy the pom.xml file and download dependencies
# This step is cached, so dependencies will only be re-downloaded if the pom.xml file changes
COPY pom.xml .
RUN mvn dependency:go-offline

# Copy the project source code and build the application
COPY src ./src
RUN mvn package -DskipTests

# Stage 2: Create the runtime image
FROM openjdk:17-slim

# Set a non-root user for security purposes
RUN groupadd -r javauser && useradd -r -g javauser javauser
USER javauser

# Set the application's working directory
WORKDIR /app

# Copy the built application from the build stage to the current stage
COPY --from=build /app/target/*.jar app.jar

# Specify the command to run the application
CMD ["java", "-jar", "app.jar"]
