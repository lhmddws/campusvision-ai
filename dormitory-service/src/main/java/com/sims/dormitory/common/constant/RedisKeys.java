package com.sims.dormitory.common.constant;

public final class RedisKeys {

    private RedisKeys() {}

    public static String studentStatus(String studentId) {
        return "dorm:student:" + studentId + ":status";
    }

    public static String buildingStudents(Long buildingId) {
        return "dorm:building:" + buildingId + ":students";
    }

    public static String buildingStatus(Long buildingId) {
        return "dorm:building:" + buildingId + ":status";
    }

    public static String eventProcessed(String eventId) {
        return "dorm:event:processed:" + eventId;
    }

    public static String todayReport(Long buildingId) {
        return "dorm:report:today:" + buildingId;
    }

    public static final String CONFIG = "dorm:config";

    public static String alertCooldown(String type) {
        return "dorm:alert:cooldown:" + type;
    }
}
