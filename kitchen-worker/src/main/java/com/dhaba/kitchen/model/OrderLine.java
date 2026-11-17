package com.dhaba.kitchen.model;

/** One line of an order as stored in order_items. */
public record OrderLine(int menuItemId, String name, int quantity, int lineTotalCents) {
}
