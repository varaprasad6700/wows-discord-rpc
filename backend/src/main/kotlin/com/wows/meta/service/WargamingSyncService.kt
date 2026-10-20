package com.wows.meta.service

import com.wows.meta.dto.WargamingApiResponse
import com.wows.meta.model.ShipEntity
import com.wows.meta.repository.ShipRepository
import com.wows.meta.util.RomanNumeral
import org.slf4j.LoggerFactory
import org.springframework.beans.factory.annotation.Value
import org.springframework.scheduling.annotation.Async
import org.springframework.stereotype.Service
import org.springframework.web.client.RestTemplate
import java.time.Instant

@Service
class WargamingSyncService(
    private val shipRepository: ShipRepository,
    @Value("\${wargaming.api.application-id:demo_app_id}")
    private val applicationId: String,
    @Value("\${wargaming.api.base-url:https://api.worldofwarships.eu}")
    private val baseUrl: String
) {
    private val logger = LoggerFactory.getLogger(javaClass)
    private val restTemplate = RestTemplate()

    @Async
    fun syncAllShipsAsync(): String {
        return syncAllShips()
    }

    fun syncAllShips(): String {
        logger.info("Starting Wargaming ship encyclopedia sync...")
        var page = 1
        var totalPages = 1
        var syncedCount = 0

        try {
            while (page <= totalPages) {
                val url = "$baseUrl/wows/encyclopedia/ships/?application_id=$applicationId&page_no=$page&limit=100&fields=ship_id,name,tier,type,nation,images.small"
                val response = restTemplate.getForObject(url, WargamingApiResponse::class.java)

                if (response == null || response.status != "ok" || response.data == null) {
                    logger.warn("Received empty or error response on page $page: ${response?.status}")
                    break
                }

                totalPages = response.meta.pageTotal
                for ((_, shipData) in response.data) {
                    val entity = ShipEntity(
                        shipId = shipData.shipId,
                        name = shipData.name,
                        tier = shipData.tier,
                        romanTier = RomanNumeral.toRoman(shipData.tier),
                        shipClass = shipData.type,
                        nation = shipData.nation,
                        iconUrl = shipData.images?.small,
                        lastUpdated = Instant.now()
                    )
                    shipRepository.save(entity)
                    syncedCount++
                }

                page++
            }
            logger.info("Ship encyclopedia sync finished. Synced $syncedCount ships across $totalPages pages.")
            return "SYNCED_$syncedCount"
        } catch (e: Exception) {
            logger.error("Error during ship encyclopedia sync: ${e.message}", e)
            return "SYNC_FAILED"
        }
    }

    fun fetchAndSaveShip(shipId: Long): ShipEntity? {
        try {
            val url = "$baseUrl/wows/encyclopedia/ships/?application_id=$applicationId&ship_id=$shipId&fields=ship_id,name,tier,type,nation,images.small"
            val response = restTemplate.getForObject(url, WargamingApiResponse::class.java)
            if (response != null && response.status == "ok" && response.data != null) {
                val shipData = response.data[shipId.toString()] ?: return null
                val entity = ShipEntity(
                    shipId = shipData.shipId,
                    name = shipData.name,
                    tier = shipData.tier,
                    romanTier = RomanNumeral.toRoman(shipData.tier),
                    shipClass = shipData.type,
                    nation = shipData.nation,
                    iconUrl = shipData.images?.small,
                    lastUpdated = Instant.now()
                )
                return shipRepository.save(entity)
            }
        } catch (e: Exception) {
            logger.warn("Failed on-demand ship lookup for $shipId: ${e.message}")
        }
        return null
    }
}
