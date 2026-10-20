package com.wows.meta.controller

import com.wows.meta.model.ShipEntity
import com.wows.meta.repository.ShipRepository
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.http.MediaType
import org.springframework.test.context.ActiveProfiles
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.get
import org.springframework.test.web.servlet.post

@SpringBootTest
@AutoConfigureMockMvc
@ActiveProfiles("test")
class ShipControllerTest @Autowired constructor(
    val mockMvc: MockMvc,
    val shipRepository: ShipRepository
) {
    @BeforeEach
    fun setup() {
        shipRepository.deleteAll()
        shipRepository.save(
            ShipEntity(
                shipId = 4181230000L,
                name = "Yamato",
                tier = 10,
                romanTier = "X",
                shipClass = "Battleship",
                nation = "japan",
                iconUrl = "https://glossary-wows-global.gcdn.co/icons/ship/japan/4181230000.png"
            )
        )
    }

    @Test
    fun testGetShip_Success() {
        mockMvc.get("/api/v1/ships/4181230000")
            .andExpect {
                status { isOk() }
                content { contentType(MediaType.APPLICATION_JSON) }
                jsonPath("$.shipId") { value(4181230000L) }
                jsonPath("$.name") { value("Yamato") }
                jsonPath("$.tier") { value(10) }
                jsonPath("$.romanTier") { value("X") }
                jsonPath("$.shipClass") { value("Battleship") }
            }
    }

    @Test
    fun testGetShip_NotFound() {
        mockMvc.get("/api/v1/ships/9999999999")
            .andExpect {
                status { isNotFound() }
            }
    }

    @Test
    fun testSyncShips_Trigger() {
        mockMvc.post("/api/v1/ships/sync")
            .andExpect {
                status { isOk() }
                jsonPath("$.status") { value("SYNC_TRIGGERED") }
            }
    }
}
