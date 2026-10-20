package com.wows.meta

import org.springframework.boot.autoconfigure.SpringBootApplication
import org.springframework.boot.runApplication

@SpringBootApplication
class WowsMetaServiceApplication

fun main(args: Array<String>) {
    runApplication<WowsMetaServiceApplication>(*args)
}
