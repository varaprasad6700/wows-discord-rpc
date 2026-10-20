package com.wows.meta.repository

import com.wows.meta.model.ShipEntity
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.autoconfigure.orm.jpa.DataJpaTest
import org.springframework.test.context.ActiveProfiles

@DataJpaTest
@ActiveProfiles("test")
class ShipRepositoryTest @Autowired constructor(
    val shipRepository: ShipRepository
) {
    @Test
    fun testSaveAndFindShip() {
        val ship = ShipEntity(
            shipId = 4181230000L,
            name = "Yamato",
            tier = 10,
            romanTier = "X",
            shipClass = "Battleship",
            nation = "japan",
            iconUrl = "https://glossary-wows-global.gcdn.co/icons/ship/japan/4181230000.png"
        )
        shipRepository.save(ship)

        val found = shipRepository.findById(4181230000L)
        assertTrue(found.isPresent)
        assertEquals("Yamato", found.get().name)
        assertEquals("X", found.get().romanTier)
        assertEquals("Battleship", found.get().shipClass)
    }
}
