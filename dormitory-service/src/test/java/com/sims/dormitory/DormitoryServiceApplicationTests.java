package com.sims.dormitory;

import com.sims.dormitory.consumer.EventConsumer;
import com.sims.dormitory.repository.DormAlertMapper;
import com.sims.dormitory.repository.DormAttendanceRecordMapper;
import com.sims.dormitory.repository.DormBuildingMapper;
import com.sims.dormitory.repository.DormCameraMapper;
import com.sims.dormitory.repository.DormConfigMapper;
import com.sims.dormitory.repository.DormDailySummaryMapper;
import com.sims.dormitory.repository.DormEventLogMapper;
import com.sims.dormitory.repository.DormNightlyReportMapper;
import com.sims.dormitory.repository.DormRoomMapper;
import com.sims.dormitory.repository.DormStrangerRecordMapper;
import com.sims.dormitory.repository.DormStudentMapper;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.boot.SpringBootConfiguration;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;
import org.springframework.boot.autoconfigure.kafka.KafkaAutoConfiguration;
import org.springframework.boot.autoconfigure.kafka.KafkaProperties;
import org.springframework.boot.autoconfigure.data.redis.RedisAutoConfiguration;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.FilterType;
import org.springframework.context.annotation.Import;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.ActiveProfiles;

import static org.mockito.Mockito.mock;

/**
 * Context load smoke test. Excludes all infrastructure auto-configs
 * (DB, Kafka, Redis, MyBatis) and provides mock beans for the mapper
 * interfaces so the service layer can load successfully.
 * The {@link EventConsumer} is excluded from scanning because its
 * {@code @KafkaListener} methods require a live Kafka broker.
 */
@SpringBootTest(classes = DormitoryServiceApplicationTests.TestApp.class)
@Import(DormitoryServiceApplicationTests.TestConfig.class)
@ActiveProfiles("test")
@DisplayName("DormitoryServiceApplication Context Load Test")
class DormitoryServiceApplicationTests {

    @SpringBootConfiguration
    @EnableAutoConfiguration(exclude = {
        DataSourceAutoConfiguration.class,
        KafkaAutoConfiguration.class,
        RedisAutoConfiguration.class,
        com.baomidou.mybatisplus.autoconfigure.MybatisPlusAutoConfiguration.class,
        com.baomidou.mybatisplus.autoconfigure.MybatisPlusLanguageDriverAutoConfiguration.class,
        com.baomidou.mybatisplus.autoconfigure.MybatisPlusInnerInterceptorAutoConfiguration.class,
        com.baomidou.mybatisplus.autoconfigure.IdentifierGeneratorAutoConfiguration.class,
        com.baomidou.mybatisplus.autoconfigure.DdlAutoConfiguration.class
    })
    @ComponentScan(
        basePackages = "com.sims.dormitory",
        excludeFilters = {
            @ComponentScan.Filter(
                type = FilterType.ASSIGNABLE_TYPE,
                value = EventConsumer.class
            ),
            @ComponentScan.Filter(
                type = FilterType.ASSIGNABLE_TYPE,
                value = DormitoryServiceApplication.class
            )
        }
    )
    static class TestApp {
    }

    @TestConfiguration
    static class TestConfig {
        @Bean
        KafkaProperties kafkaProperties() {
            return new KafkaProperties();
        }

        @Bean
        RedisConnectionFactory redisConnectionFactory() {
            return mock(RedisConnectionFactory.class);
        }

        @Bean
        DormCameraMapper dormCameraMapper() { return mock(DormCameraMapper.class); }

        @Bean
        DormEventLogMapper dormEventLogMapper() { return mock(DormEventLogMapper.class); }

        @Bean
        DormStudentMapper dormStudentMapper() { return mock(DormStudentMapper.class); }

        @Bean
        DormAlertMapper dormAlertMapper() { return mock(DormAlertMapper.class); }

        @Bean
        DormStrangerRecordMapper dormStrangerRecordMapper() { return mock(DormStrangerRecordMapper.class); }

        @Bean
        DormBuildingMapper dormBuildingMapper() { return mock(DormBuildingMapper.class); }

        @Bean
        DormConfigMapper dormConfigMapper() { return mock(DormConfigMapper.class); }

        @Bean
        DormAttendanceRecordMapper dormAttendanceRecordMapper() { return mock(DormAttendanceRecordMapper.class); }

        @Bean
        DormDailySummaryMapper dormDailySummaryMapper() { return mock(DormDailySummaryMapper.class); }

        @Bean
        DormNightlyReportMapper dormNightlyReportMapper() { return mock(DormNightlyReportMapper.class); }

        @Bean
        DormRoomMapper dormRoomMapper() { return mock(DormRoomMapper.class); }

        @Bean
        JdbcTemplate jdbcTemplate() { return mock(JdbcTemplate.class); }
    }

    @Test
    @DisplayName("application context loads successfully")
    void contextLoads() {
    }
}
