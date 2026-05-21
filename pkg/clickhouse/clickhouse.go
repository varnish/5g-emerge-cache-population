package clickhouse

import (
	"fmt"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
)

func GetSql(cfg config.Config) string {
	return fmt.Sprintf(`WITH
			toUInt64(toUnixTimestamp64Nano(toDateTime64(now(), 9))) AS now_ns,
			%d AS interval_ns,
			now_ns - interval_ns AS t_5m_ns,
			now_ns - 2 * interval_ns AS t_10m_ns,
			now_ns - 3 * interval_ns AS t_15m_ns,
			now_ns - 4 * interval_ns AS t_20m_ns,
			'%s' AS service_name,
			'client' AS varnish_side
		SELECT
			title,
			sum(watch_duration) as total_watch_duration
		FROM (
			SELECT
				attributes_string['video.title.name'] AS title,
				sum(toFloat64(attributes_string['video.duration'])) AS watch_duration
			FROM signoz_logs.distributed_logs_v2
			WHERE
				toString(resource.service.name) = service_name
				AND trim(attributes_string['varnish.side']) = varnish_side
				AND trim(attributes_string['video.title.name']) != ''
				AND trim(attributes_string['video.duration']) != ''
				AND observed_timestamp >= t_5m_ns
				AND observed_timestamp <  now_ns
			GROUP BY title

			UNION ALL

			SELECT
				attributes_string['video.title.name'] AS title,
				sum(toFloat64(attributes_string['video.duration']) * 0.5) AS watch_duration
			FROM signoz_logs.distributed_logs_v2
			WHERE
				toString(resource.service.name) = service_name
				AND trim(attributes_string['varnish.side']) = varnish_side
				AND trim(attributes_string['video.title.name']) != ''
				AND trim(attributes_string['video.duration']) != ''
				AND observed_timestamp >= t_10m_ns
				AND observed_timestamp <  t_5m_ns
			GROUP BY title

			UNION ALL

			SELECT
				attributes_string['video.title.name'] AS title,
				sum(toFloat64(attributes_string['video.duration']) * 0.25) AS watch_duration
			FROM signoz_logs.distributed_logs_v2
			WHERE
				toString(resource.service.name) = service_name
				AND trim(attributes_string['varnish.side']) = varnish_side
				AND trim(attributes_string['video.title.name']) != ''
				AND trim(attributes_string['video.duration']) != ''
				AND observed_timestamp >= t_15m_ns
				AND observed_timestamp <  t_10m_ns
			GROUP BY title

			UNION ALL

			SELECT
				attributes_string['video.title.name'] AS title,
				sum(toFloat64(attributes_string['video.duration']) * 0.125) AS watch_duration
			FROM signoz_logs.distributed_logs_v2
			WHERE
				toString(resource.service.name) = service_name
				AND trim(attributes_string['varnish.side']) = varnish_side
				AND trim(attributes_string['video.title.name']) != ''
				AND trim(attributes_string['video.duration']) != ''
				AND observed_timestamp >= t_20m_ns
				AND observed_timestamp <  t_15m_ns
			GROUP BY title
		) intervals
		GROUP by title
		ORDER BY sum(watch_duration) DESC`,
		cfg.QueryRange.Nanoseconds(),
		cfg.SignozOtelQueryService,
	)
}
