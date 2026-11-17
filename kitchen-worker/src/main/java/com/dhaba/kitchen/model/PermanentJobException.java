package com.dhaba.kitchen.model;

/** A job that can never succeed (bad data, missing order). It is dead-lettered without retries. */
public class PermanentJobException extends RuntimeException {

    private static final long serialVersionUID = 1L;

    public PermanentJobException(String message) {
        super(message);
    }
}
