package com.dhaba.kitchen;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;
import org.springframework.scheduling.annotation.EnableScheduling;

/** Entry point. The worker has no HTTP port: it consumes Redis jobs and writes PostgreSQL. */
@SpringBootApplication
@ConfigurationPropertiesScan
@EnableScheduling
public class KitchenWorkerApplication {

    protected KitchenWorkerApplication() {
    }

    public static void main(String[] args) {
        SpringApplication.run(KitchenWorkerApplication.class, args);
    }
}
