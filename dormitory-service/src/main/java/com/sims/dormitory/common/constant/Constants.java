package com.sims.dormitory.common.constant;

public interface Constants {

    String TOPIC_EVENT = "t_dorm_event";
    String TOPIC_ALERT = "t_dorm_alert";

    String CACHE_PREFIX_STUDENT = "dorm:student:";
    String CACHE_PREFIX_CAMERA = "dorm:camera:";
    String CACHE_PREFIX_BUILDING = "dorm:building:";
    String CACHE_PREFIX_EVENT = "dorm:event:processed:";

    String DATE_FORMAT = "yyyy-MM-dd";
    String DATE_TIME_FORMAT = "yyyy-MM-dd'T'HH:mm:ss";
    String TIME_ZONE = "Asia/Shanghai";

    int DEFAULT_PAGE = 1;
    int DEFAULT_PAGE_SIZE = 20;
    int MAX_PAGE_SIZE = 100;

    int MAX_CAMERA_COUNT = 50;

    int CACHE_STATUS_TTL_HOURS = 6;
    int DEDUP_TTL_SECONDS = 3600;
    int ALERT_COOLDOWN_SECONDS = 300;
}
