package com.wows.meta.controller

import com.wows.meta.dto.ShipResponseDto
import com.wows.meta.repository.ShipRepository
import com.wows.meta.service.WargamingSyncService
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.*

@RestController
@RequestMapping("/api/v1/ships")
class ShipController(
    private val shipRepository: ShipRepository,
    private val wargamingSyncService: WargamingSyncService
) {
    @GetMapping("/{shipId}")
    fun getShip(@PathVariable shipId: Long): ResponseEntity<ShipResponseDto> {
        val cached = shipRepository.findById(shipId)
        if (cached.isPresent) {
            val ship = cached.get()
            return ResponseEntity.ok(
                ShipResponseDto(
                    shipId = ship.shipId,
                    name = ship.name,
                    tier = ship.tier,
                    romanTier = ship.romanTier,
                    shipClass = ship.shipClass,
                    nation = ship.nation,
                    iconUrl = ship.iconUrl
                )
            )
        }

        // On-demand cache miss resolution
        val fetched = wargamingSyncService.fetchAndSaveShip(shipId)
        if (fetched != null) {
            return ResponseEntity.ok(
                ShipResponseDto(
                    shipId = fetched.shipId,
                    name = fetched.name,
                    tier = fetched.tier,
                    romanTier = fetched.romanTier,
                    shipClass = fetched.shipClass,
                    nation = fetched.nation,
                    iconUrl = fetched.iconUrl
                )
            )
        }

        return ResponseEntity.notFound().build()
    }

    @PostMapping("/sync")
    fun syncShips(): ResponseEntity<Map<String, String>> {
        wargamingSyncService.syncAllShipsAsync()
        return ResponseEntity.ok(mapOf("status" to "SYNC_TRIGGERED"))
    }
}
