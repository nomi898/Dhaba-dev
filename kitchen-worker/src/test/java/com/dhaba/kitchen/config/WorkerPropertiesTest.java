package com.dhaba.kitchen.config;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.dhaba.kitchen.TestFixtures;
import java.time.Duration;
import org.junit.jupiter.api.Test;

class WorkerPropertiesTest {

    @Test
    void backoffIsExponentialAndCapped() {
        WorkerProperties p = TestFixtures.props(false);
        assertThat(p.backoffFor(0)).isEqualTo(Duration.ofSeconds(1));
        assertThat(p.backoffFor(1)).isEqualTo(Duration.ofSeconds(2));
        assertThat(p.backoffFor(3)).isEqualTo(Duration.ofSeconds(8));
        assertThat(p.backoffFor(10)).isEqualTo(Duration.ofSeconds(10));
        assertThat(p.backoffFor(500)).isEqualTo(Duration.ofSeconds(10));
    }

    @Test
    void rejectsInvalidSettings() {
        assertThatThrownBy(() -> new WorkerProperties(" ", "v", 3, Duration.ZERO, Duration.ZERO, Duration.ZERO,
                Duration.ZERO, Duration.ZERO, "", false, Duration.ZERO, Duration.ZERO))
                .isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> new WorkerProperties("w", "v", 0, Duration.ZERO, Duration.ZERO, Duration.ZERO,
                Duration.ZERO, Duration.ZERO, "", false, Duration.ZERO, Duration.ZERO))
                .isInstanceOf(IllegalArgumentException.class);
    }
}
